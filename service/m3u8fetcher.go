package service

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	httpproxy "github.com/fopina/net-proxy-httpconnect/proxy"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
)

// A Dialer is a means to establish a connection.
// Custom dialers should also implement ContextDialer.
type Dialer interface {
	// Dial connects to the given address via the proxy.
	Dial(network, addr string) (c net.Conn, err error)
}

var errNoMatchFound error = errors.New("This channel is not currently live")

func GetLiveM3U8(channel *model.Channel) (*model.LiveInfo, error) {
	liveInfo, ok := global.URLCache.Load(channel.URL)
	if ok {
		return liveInfo, nil
	} else {
		log.Println("cache miss", channel.URL)
		status := GetStatus(channel.URL)
		coolDownInterval := time.Second * time.Duration(status.CoolDownMultiplier)
		if coolDownInterval > time.Minute*2 {
			coolDownInterval = time.Minute * 2
		}
		if time.Now().Sub(status.Time) > coolDownInterval {
			if liveInfo, err := UpdateURLCacheSingle(channel, true); err == nil {
				return liveInfo, nil
			} else {
				if status.CoolDownMultiplier < 1024 {
					status.CoolDownMultiplier *= 2
				}
				return nil, err
			}
		} else {
			return nil, errors.New("parser cooling down")
		}
	}
}

func isValidM3U(content string) bool {
	content = strings.TrimSpace(string(content))
	return strings.HasPrefix(content, "#EXTM3U")
}

// returns: content, updated m3u8url (if needed), error
// func GetM3U8Content(c *gin.Context, ChannelURL string, liveM3U8 string, ProxyUrl string, Parser string, flags ...bool) (string, string, error) {
func GetM3U8Content(c *gin.Context, Channel *model.Channel, liveInfo *model.LiveInfo, flags ...bool) (string, string, error) {
	// parse the optional flags
	retryFlag := false
	if len(flags) > 0 {
		retryFlag = flags[0]
	}

	retry := func(bodyString string, err error) (string, string, error) {
		newUrl := liveInfo.LiveUrl
		chStatus := GetStatus(Channel.URL)
		if !retryFlag && chStatus.RetryCount < MaxRetryCount {
			// this channel was previously running ok, we give it a chance to reparse itself
			log.Println(Channel.URL, "is unhealthy, doing a reparse...")
			if li, err := UpdateURLCacheSingle(Channel, false); err == nil {
				UpdateStatus(Channel.URL, Warning, "Unhealthy")
				bodyString, newUrl, err = GetM3U8Content(c, Channel, li, true)
				if err == nil {
					log.Println(Channel.URL, "is back online now")
					UpdateStatus(Channel.URL, Ok, "Live!") // revert our temporary warning status to ok
				} else {
					log.Println(Channel.URL, "is still unhealthy, giving up, currently points to", li.LiveUrl)
				}
				// if error still persists after a reparse, keep our warning status so that we won't endlessly reparse the same feed
			}
		}
		return bodyString, newUrl, err
	}

	li, _ := global.URLCache.Load(Channel.URL)

	var dialer Dialer
	dialer = &net.Dialer{
		Timeout: global.HttpClientTimeout,
	}
	if Channel.ProxyUrl != "" {
		if u, err := url.Parse(Channel.ProxyUrl); err == nil {
			if d, err := proxy.FromURL(u, dialer); err == nil {
				dialer = d
			}
		}
	}
	client := http.Client{
		Timeout:   global.HttpClientTimeout,
		Transport: global.TransportWithProxy(""),
		Jar:       global.CookieJar,
	}
	req, err := http.NewRequest(http.MethodGet, liveInfo.LiveUrl, nil)
	if err != nil {
		log.Println(err)
		return "", liveInfo.LiveUrl, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	/* remove parameter passthrough as it may conflict with some servers
	queries := c.Request.URL.Query()
	reqQuery := req.URL.Query()
	for key, values := range queries {
		if strings.HasPrefix(key, "header") || slices.Contains([]string{"k", "c", "token"}, key) {
			continue
		}
		for _, value := range values {
			reqQuery.Add(key, value)
		}
	}
	req.URL.RawQuery = reqQuery.Encode()
	*/

	// allow plugins to decorate the m3u8 url
	if p, err := GetPlugin(Channel.Parser); err == nil {
		if transformer, ok := p.(Transformer); ok {
			if li != nil {
				transformer.Transform(req, li)
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", liveInfo.LiveUrl, err
	}

	bodyString := ""
	defer global.CloseBody(resp)
	// retry on server status error
	if resp.StatusCode != http.StatusOK {
		//body, _ := io.ReadAll(resp.Body)
		//log.Println("visiting", liveM3U8, req.URL, "error:", string(body))
		return retry(bodyString, errors.New(fmt.Sprintf("Server response: HTTP %d", resp.StatusCode)))
	}

	// check if the response is in a correct mime-type and with correct content.
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	isValid := resp.ContentLength < 10*1024*1024 && (strings.Contains(contentType, "mpegurl") || strings.Contains(contentType, "text"))
	if isValid {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", liveInfo.LiveUrl, err
		}
		bodyString = strings.TrimSpace(string(bodyBytes))
		isValid = isValidM3U(bodyString)
	}

	// valid check passed.
	if isValid {
		// do custom health checks
		// retry on custom health check error
		if p, err := GetPlugin(Channel.Parser); err == nil {
			if checker, ok := p.(HealthCheck); ok {
				healthErr := checker.Check(bodyString, li)
				if healthErr != nil {
					return retry(bodyString, healthErr)
				}
			}
		}
	} else {
		UpdateStatus(Channel.URL, Warning, "Url is not a live stream")
		duration, err := GetVideoDuration(Channel.URL)
		if err == nil && duration > 0 {
			log.Println(Channel.URL, "duration is", duration)
			bodyString = fmt.Sprintf("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%.0f\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:%.4f, video\n%s\n#EXT-X-ENDLIST", duration, duration, liveInfo.LiveUrl)
		} else {
			log.Println("failed to get duration", err.Error())
			bodyString = "#EXTM3U\n#EXTINF:-1, video\n#EXT-X-PLAYLIST-TYPE:VOD\n" + liveInfo.LiveUrl + "\n#EXT-X-ENDLIST" // make a fake m3u8 pointing to the target
		}
	}
	return bodyString, liveInfo.LiveUrl, nil
}

func RealLiveM3U8(channel *model.Channel) (*model.LiveInfo, error) {
	Parser := channel.Parser
	if Parser == "" {
		Parser = "youtube" // backward compatible with old database, use youtube parser by default
	}
	if p, err := GetPlugin(Parser); err == nil {
		//if liveInfo, ok := global.URLCache.Load(channel.URL); ok {
		//	//return p.Parse(channel.URL, channel.ProxyUrl, liveInfo.ExtraInfo)
		//	return p.Parse(channel, liveInfo)
		//}
		if d, ok := p.(Detector); ok {
			newPlugin, err := d.Detect(channel)
			if err != nil {
				return nil, err
			}
			if p, err = GetPlugin(newPlugin); err != nil {
				return nil, err
			}
			channel.Parser = newPlugin
		}
		return p.Parse(channel, &model.LiveInfo{})
	} else {
		return nil, err
	}
}

func init() {
	httpproxy.RegisterSchemes()
}
