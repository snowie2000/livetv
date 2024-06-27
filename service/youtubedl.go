package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	httpproxy "github.com/fopina/net-proxy-httpconnect/proxy"

	"github.com/zjyl1994/livetv/global"
	"github.com/zjyl1994/livetv/model"
	"github.com/zjyl1994/livetv/plugin"
)

// A Dialer is a means to establish a connection.
// Custom dialers should also implement ContextDialer.
type Dialer interface {
	// Dial connects to the given address via the proxy.
	Dial(network, addr string) (c net.Conn, err error)
}

var errNoMatchFound error = errors.New("This channel is not currently live")

const DefaultUserAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

func GetLiveM3U8(youtubeURL string, proxyUrl string, Parser string) (*model.LiveInfo, error) {
	liveInfo, ok := global.URLCache.Load(youtubeURL)
	if ok {
		return liveInfo, nil
	} else {
		log.Println("cache miss", youtubeURL)
		status := GetStatus(youtubeURL)
		coolDownInterval := time.Second * time.Duration(status.CoolDownMultiplier)
		if coolDownInterval > time.Minute*2 {
			coolDownInterval = time.Minute * 2
		}
		if time.Now().Sub(status.Time) > coolDownInterval {
			if liveInfo, err := UpdateURLCacheSingle(youtubeURL, proxyUrl, Parser, true); err == nil {
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

// returns: content, updated m3u8url (if needed), error
func GetM3U8Content(ChannelURL string, liveM3U8 string, ProxyUrl string, Parser string, flags ...bool) (string, string, error) {
	// parse the optional flags
	retryFlag := false
	if len(flags) > 0 {
		retryFlag = flags[0]
	}

	retry := func(bodyString string, err error) (string, string, error) {
		newUrl := liveM3U8
		chStatus := GetStatus(ChannelURL)
		if !retryFlag && chStatus.RetryCount < MaxRetryCount {
			// this channel was previously running ok, we give it a chance to reparse itself
			log.Println(ChannelURL, "is unhealthy, doing a reparse...")
			if li, err := UpdateURLCacheSingle(ChannelURL, ProxyUrl, Parser, false); err == nil {
				UpdateStatus(ChannelURL, Warning, "Unhealthy")
				bodyString, newUrl, err = GetM3U8Content(ChannelURL, li.LiveUrl, ProxyUrl, Parser, true)
				if err == nil {
					log.Println(ChannelURL, "is back online now")
					UpdateStatus(ChannelURL, Ok, "Live!") // revert our temporary warning status to ok
				} else {
					log.Println(ChannelURL, "is still unhealthy, giving up, currently pointing to", liveM3U8)
				}
				// if error still persists after a reparse, keep our warning status so that we won't endlessly reparse the same feed
			}
		}
		return bodyString, newUrl, err
	}

	li, _ := global.URLCache.Load(ChannelURL)

	var dialer Dialer
	dialer = &net.Dialer{
		Timeout: global.HttpClientTimeout,
	}
	if ProxyUrl != "" {
		if u, err := url.Parse(ProxyUrl); err == nil {
			if d, err := proxy.FromURL(u, dialer); err == nil {
				dialer = d
			}
		}
	}
	client := http.Client{
		Timeout: global.HttpClientTimeout,
		Transport: &http.Transport{
			Dial:            dialer.Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequest(http.MethodGet, liveM3U8, nil)
	if err != nil {
		log.Println(err)
		return "", liveM3U8, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	// allow plugins to decorate the m3u8 url
	if p, err := plugin.GetPlugin(Parser); err == nil {
		if transformer, ok := p.(plugin.Transformer); ok {
			if li != nil {
				transformer.Transform(req, li)
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", liveM3U8, err
	}

	bodyString := ""
	defer resp.Body.Close()
	// retry on server status error
	if resp.StatusCode != http.StatusOK {
		return retry(bodyString, errors.New(fmt.Sprintf("Server response: HTTP %d", resp.StatusCode)))
	}

	if resp.ContentLength < 10*1024*1024 && strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "mpegurl") {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", liveM3U8, err
		}
		bodyString = string(bodyBytes)
		// retry on custom health check error
		if p, err := plugin.GetPlugin(Parser); err == nil {
			if checker, ok := p.(plugin.HealthCheck); ok {
				healthErr := checker.Check(bodyString, li)
				if healthErr != nil {
					return retry(bodyString, healthErr)
				}
			}
		}
	} else {
		UpdateStatus(ChannelURL, Warning, "Url is not a live stream")
		duration, err := GetVideoDuration(ChannelURL)
		if err == nil && duration > 0 {
			log.Println(ChannelURL, "duration is", duration)
			bodyString = fmt.Sprintf("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:%.0f\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXT-X-MEDIA-SEQUENCE:0\n#EXTINF:%.4f, video\n%s\n#EXT-X-ENDLIST", duration, duration, liveM3U8)
		} else {
			log.Println("failed to get duration", err.Error())
			bodyString = "#EXTM3U\n#EXTINF:-1, video\n#EXT-X-PLAYLIST-TYPE:VOD\n" + liveM3U8 + "\n#EXT-X-ENDLIST" // make a fake m3u8 pointing to the target
		}
	}
	return bodyString, liveM3U8, nil
}

func RealLiveM3U8(liveUrl string, proxyUrl string, Parser string) (*model.LiveInfo, error) {
	if Parser == "" {
		Parser = "youtube" // backward compatible with old database, use youtube parser by default
	}
	if p, err := plugin.GetPlugin(Parser); err == nil {
		if liveInfo, ok := global.URLCache.Load(liveUrl); ok {
			return p.Parse(liveUrl, proxyUrl, liveInfo.ExtraInfo)
		}
		return p.Parse(liveUrl, proxyUrl, "")
	} else {
		return nil, err
	}
}

func init() {
	httpproxy.RegisterSchemes()
}
