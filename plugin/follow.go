// follow location
package plugin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
)

type URLM3U8Parser struct {
	DirectM3U8Parser
}

func (p *URLM3U8Parser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: global.TransportWithProxy(proxyUrl),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest("GET", liveUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	redir := resp.Header.Get("Location")
	if redir != "" {
		// unpack previousExtraInfo
		var pei UrlInfo
		json.Unmarshal([]byte(previousExtraInfo), &pei)
		if pei.RedirectCounter > 5 {
			return nil, errors.New("Too many redirections")
		}

		var ui UrlInfo
		decoder := json.NewDecoder(resp.Body)
		if decoder.Decode(&ui) == nil && len(ui.Headers) > 0 {
			ui.RedirectCounter = pei.RedirectCounter + 1
			js, _ := json.Marshal(ui)
			previousExtraInfo = string(js) // write headers info to extraInfo
		}
		info, err := p.Parse(redir, proxyUrl, previousExtraInfo) // recursive call the parser to follow redirections
		if err == nil && info != nil {
			info.Logo = ui.Logo
		}
		return info, err
	}
	// this is a direct m3u8 url, let's parse the content
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "mpegurl") {
		return p.DirectM3U8Parser.Parse(liveUrl, proxyUrl, previousExtraInfo, resp.Body)
	}
	return nil, errors.New("Invalid feed: " + resp.Header.Get("Content-Type"))
}

func init() {
	registerPlugin("http", &URLM3U8Parser{}, 0)
}
