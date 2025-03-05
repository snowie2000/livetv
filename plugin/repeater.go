// repeater
// instead of providing the m3u8 content by ourselves, repeater use http 302 to redirect clients to the original url so further requests will be fired to us
package plugin

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
)

type RepeaterParser struct{}

func (p *RepeaterParser) Host(c *gin.Context, info *model.LiveInfo) error {
	c.Redirect(http.StatusFound, info.LiveUrl)
	return nil
}

func (p *RepeaterParser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
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
		Transport: global.TransportWithProxy(proxyUrl),
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

	defer global.CloseBody(resp)
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
	registerPlugin("repeater", &RepeaterParser{}, 6)
}
