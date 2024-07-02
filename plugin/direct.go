// direct
package plugin

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
)

type DirectM3U8Parser struct{}

func (p *DirectM3U8Parser) Transform(req *http.Request, info *model.LiveInfo) error {
	var ui UrlInfo
	json.Unmarshal([]byte(info.ExtraInfo), &ui)
	for v, k := range ui.Headers {
		req.Header.Add(v, k)
	}
	return nil
}

func (p *DirectM3U8Parser) TransformTs(rawLink string, tsLink string, info *model.LiveInfo) string {
	var ui UrlInfo
	json.Unmarshal([]byte(info.ExtraInfo), &ui)
	u, err := url.Parse(tsLink)
	if err == nil {
		q := u.Query()
		for v, k := range ui.Headers {
			q.Add("header"+v, k)
		}
		u.RawQuery = q.Encode()
		return u.String()
	}
	return tsLink
}

func (p *DirectM3U8Parser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
	u, err := url.Parse(liveUrl)
	if err != nil {
		return nil, err
	}
	// return non http protocol directly
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		li := &model.LiveInfo{}
		li.LiveUrl = liveUrl
		return li, nil
	}

	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: transportWithProxy(proxyUrl),
	}
	req, err := http.NewRequest("GET", liveUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	// allow adding custom transformations
	p.Transform(req, &model.LiveInfo{
		ExtraInfo: previousExtraInfo,
	})
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	// the link itself is a valid M3U8
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "mpegurl") {
		log.Println(liveUrl, "is a valid url")
		liveUrl, err := bestFromMasterPlaylist(liveUrl, proxyUrl, resp.Body) // extract the best quality live url from the master playlist
		if err == nil {
			li := &model.LiveInfo{}
			if !global.IsValidURL(liveUrl) {
				liveUrl = global.GetBaseURL(liveUrl) + liveUrl
			}
			li.LiveUrl = liveUrl
			li.ExtraInfo = previousExtraInfo
			return li, nil
		}
	}
	return nil, NoMatchFeed
}

func init() {
	registerPlugin("direct", &DirectM3U8Parser{})
}
