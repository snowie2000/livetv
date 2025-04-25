// follow location
package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
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
		Jar: global.CookieJar,
	}
	req, err := http.NewRequest("GET", liveUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	p.Transform(req, &model.LiveInfo{
		ExtraInfo: previousExtraInfo,
	})
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer global.CloseBody(resp)
	redir := resp.Header.Get("Location")
	// unpack previousExtraInfo
	var pei UrlInfo
	json.Unmarshal([]byte(previousExtraInfo), &pei)
	if redir != "" {
		if pei.RedirectCounter > 5 {
			return nil, errors.New("Too many redirections")
		}

		// recreate full url for relative redirections
		if !global.IsValidURL(redir) {
			redir = global.MergeUrl(global.GetBaseURL(liveUrl), redir)
		}

		var ui UrlInfo
		decoder := json.NewDecoder(resp.Body)
		if decoder.Decode(&ui) != nil {
			ui = pei
		}

		ui.RedirectCounter = pei.RedirectCounter + 1
		js, _ := json.Marshal(ui)
		previousExtraInfo = string(js)                           // write headers info to extraInfo
		info, err := p.Parse(redir, proxyUrl, previousExtraInfo) // recursive call the parser to follow redirections
		if err == nil && info != nil {
			info.Logo = ui.Logo
		}
		return info, err
	} else {
		pei.RedirectCounter = 0 // reset counter on a successful parse
	}
	// this is a direct m3u8 url, let's parse the content
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "mpegurl") {
		js, _ := json.Marshal(pei)
		previousExtraInfo = string(js)
		return p.DirectM3U8Parser.Parse(liveUrl, proxyUrl, previousExtraInfo, resp.Body)
	} else {
		if strings.Contains(contentType, "text") {
			content := &bytes.Buffer{}
			io.Copy(content, resp.Body)
			if li, err := p.DirectM3U8Parser.Parse(liveUrl, proxyUrl, previousExtraInfo, content); err == nil {
				return li, err
			} else {
				log.Println("Server error response:", content.String())
			}
		}
	}
	return nil, errors.New("Invalid feed: " + contentType)
}

func init() {
	registerPlugin("http", &URLM3U8Parser{}, 0)
}
