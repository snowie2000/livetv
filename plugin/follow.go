// follow location
package plugin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/snowie2000/livetv/model"
)

type URLM3U8Parser struct {
	DirectM3U8Parser
}

func (p *URLM3U8Parser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: transportWithProxy(proxyUrl),
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
	var ui UrlInfo
	decoder := json.NewDecoder(resp.Body)
	if decoder.Decode(&ui) == nil && len(ui.Headers) > 0 {
		js, _ := json.Marshal(ui)
		previousExtraInfo = string(js) // write headers info to extraInfo
	}

	redir := resp.Header.Get("Location")
	if redir == "" {
		return nil, NoMatchFeed
	}
	info, err := p.DirectM3U8Parser.Parse(redir, proxyUrl, previousExtraInfo)
	if err == nil && info != nil {
		info.Logo = ui.Logo
	}
	return info, err
}

func init() {
	registerPlugin("httpRedirect", &URLM3U8Parser{})
}
