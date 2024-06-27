// plugins
package plugin

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	httpproxy "github.com/fopina/net-proxy-httpconnect/proxy"
	freq "github.com/imroc/req/v3"
	"golang.org/x/net/proxy"

	"github.com/dlclark/regexp2"
	"github.com/zjyl1994/livetv/global"
	"github.com/zjyl1994/livetv/model"

	"github.com/grafov/m3u8"
)

// plugin parser interface
type Plugin interface {
	Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (info *model.LiveInfo, error error)
}

// transform the request before getM3U8content
type Transformer interface {
	Transform(req *http.Request, info *model.LiveInfo) error
}

// do a healthcheck when GetM3U8Content returned
type HealthCheck interface {
	Check(content string, info *model.LiveInfo) error
}

// host a live feed directly instead of generating a m3u8 playlist
type FeedHost interface {
	Host(c *gin.Context, info *model.LiveInfo) error
}

// transform the tsproxy link
type TsTransformer interface {
	TransformTs(rawLink string, tsLink string, info *model.LiveInfo) string
}

type UrlInfo struct {
	Headers map[string]string `json:"headers"`
	Logo    string            `json:"logo"`
}

var (
	pluginCenter  map[string]Plugin = make(map[string]Plugin)
	NoMatchPlugin error             = errors.New("No matching plugin found")
	NoMatchFeed   error             = errors.New("This channel is not currently live")
)

const (
	DefaultUserAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

func transportWithProxy(proxyUrl string) *http.Transport {
	d := &net.Dialer{
		Timeout: global.HttpClientTimeout,
	}
	tr := &http.Transport{
		Dial:            d.Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if proxyUrl != "" {
		if u, err := url.Parse(proxyUrl); err == nil {
			if p, e := proxy.FromURL(u, d); e == nil {
				tr.Dial = p.Dial
			} else {
				log.Println("Proxy setup error:", e)
			}
		}
	}
	return tr
}

func registerPlugin(name string, parser Plugin) {
	pluginCenter[name] = parser
}

func cloudScraper(req *http.Request, proxyUrl string) (*freq.Response, error) {
	client := freq.C().ImpersonateChrome().SetCommonContentType("application/x-www-form-urlencoded; charset=UTF-8").SetCommonHeader("accept", "*/*")
	if proxyUrl != "" {
		client.SetDial(func(ctx context.Context, network, addr string) (net.Conn, error) {
			return transportWithProxy(proxyUrl).Dial(network, addr)
		})
	}
	switch req.Method {
	case http.MethodGet:
		return client.R().Get(req.URL.String())
	case http.MethodPost:
		return client.R().SetBody(req.Body).Post(req.URL.String())
	default:
		return nil, errors.New("Method not allowed")
	}

	// // Client also will need a cookie jar.
	// // client := http.Client{}
	// // cookieJar, _ := cookiejar.New(nil)
	// // client.Jar = cookieJar
	// client, _ := newTlsClient()
	// req.Header = http.Header{
	// 	// "sec-ch-ua":        {`"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"`},
	// 	"sec-ch-ua-mobile":   {`?1`},
	// 	"User-Agent":         {`Mozilla/5.0 (iPad; CPU OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) EdgiOS/121.0.2277.107 Version/16.0 Mobile/15E148 Safari/604.1`},
	// 	"Accept":             {`*/*`},
	// 	"Sec-Fetch-Site":     {`same-site`},
	// 	"Sec-Fetch-Mode":     {`cors`},
	// 	"Sec-Fetch-Dest":     {`empty`},
	// 	"Content-Type":       {"application/x-www-form-urlencoded; charset=UTF-8"},
	// 	"Accept-Encoding":    {`gzip, deflate`},
	// 	"Accept-Language":    {`en-US,en;q=0.9`},
	// 	http.HeaderOrderKey:  {"sec-ch-ua", "sec-ch-ua-mobile", "upgrade-insecure-requests", "user-agent", "accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest", "accept-encoding", "accept-language"},
	// 	http.PHeaderOrderKey: {":method", ":authority", ":scheme", ":path"},
	// }

	// return client.Do(req)
}

func bestFromMasterPlaylist(masterUrl string, proxyUrl string, content ...io.Reader) (string, error) {
	var playlist io.Reader
	if len(content) > 0 {
		playlist = content[0]
	} else {
		req, err := http.NewRequest("GET", masterUrl, nil)
		if err != nil {
			return "", err
		}
		resp, err := cloudScraper(req, proxyUrl)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.ContentLength > 10*1024*1024 {
			log.Println(masterUrl, "content too large")
			return "", errors.New("Content too large")
		}
		if !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "mpegurl") {
			log.Println(masterUrl, "content type is incorrect")
			body, _ := io.ReadAll(resp.Body)
			log.Println("body", string(body))
			return "", errors.New(resp.Header.Get("Content-Type") + " is unknown")
		}
		playlist = resp.Body
	}
	p, listType, err := m3u8.DecodeFrom(playlist, true)
	// log.Println("parsed playlist", p == nil, listType, err)
	if p == nil {
		return "", err
	}
	switch listType {
	case m3u8.MEDIA:
		{
			return masterUrl, nil
		}
	case m3u8.MASTER:
		{
			masterpl := p.(*m3u8.MasterPlaylist)
			selectedUrl := ""
			selectedBw := uint32(0)
			for _, v := range masterpl.Variants {
				if v.Bandwidth >= selectedBw {
					selectedUrl = v.URI
					selectedBw = v.Bandwidth
				}
				if v.Audio != "" {
					return masterUrl, nil // a master playlist mixed with audio and video, we have to preserve the master playlist
				}
			}
			if !global.IsValidURL(selectedUrl) {
				selectedUrl = global.MergeUrl(global.GetBaseURL(masterUrl), selectedUrl)
			}
			return selectedUrl, nil
		}
	}
	return "", errors.New("Unknown type of playlist")
}

// regex from https://stackoverflow.com/questions/5830387/how-do-i-find-all-youtube-video-ids-in-a-string-using-a-regex?lq=1
func getYouTubeVideoID(url string) string {
	regex := regexp2.MustCompile(`(?:youtu\.be\/|youtube(?:-nocookie)?\.com\S*?[^\w\s-])([\w-]{11})(?=[^\w-]|$)(?![?=&+%\w.-]*(?:['"][^<>]*>|<\/a>))[?=&+%\w.-]*`, 0)
	match, _ := regex.FindStringMatch(url)
	if match != nil && len(match.Groups()) > 0 {
		return match.Groups()[0].Captures[0].String()
	}
	return ""
}

func getYouTubeChannelID(url string) string {
	regex := regexp2.MustCompile(`youtu((\.be)|(be\..{2,5}))\/((user)|(channel)|(c)|(@))\/?([a-zA-Z0-9\-_]{1,})`, 0)
	match, _ := regex.FindStringMatch(url)
	if match != nil && len(match.Groups()) > 0 {
		return match.Groups()[9].Captures[0].String()
	}
	return ""
}

func GetPlugin(name string) (Plugin, error) {
	if p, ok := pluginCenter[name]; ok {
		return p, nil
	}
	log.Println(name, "not found")
	return nil, NoMatchPlugin
}

func GetPluginList() []string {
	list := make([]string, 0)
	for name, _ := range pluginCenter {
		list = append(list, name)
	}
	return list
}

func init() {
	httpproxy.RegisterSchemes()
}
