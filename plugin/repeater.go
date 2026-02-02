// repeater
// instead of providing the m3u8 content by ourselves, repeater use http 302 to redirect clients to the original url so further requests will be fired to us
package plugin

import (
	"github.com/snowie2000/livetv/service"
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

func (p *RepeaterParser) Host(c *gin.Context, info *model.LiveInfo, chInfo *model.Channel) error {
	c.Redirect(http.StatusFound, info.LiveUrl)
	return nil
}

func (p *RepeaterParser) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	u, err := url.Parse(channel.URL)
	if err != nil {
		return nil, err
	}
	// return non http protocol directly
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		li := &model.LiveInfo{}
		li.LiveUrl = channel.URL
		return li, nil
	}

	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: global.TransportWithProxy(channel.ProxyUrl),
		Jar:       global.CookieJar,
	}
	req, err := http.NewRequest("GET", channel.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", service.DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	liveUrl := resp.Request.URL.String() // replace source url with potentially redirected url
	defer global.CloseBody(resp)
	// the link itself is a valid M3U8
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "mpegurl") {
		log.Println(liveUrl, "is a valid url")
		liveUrl, err := service.BestFromMasterPlaylist(liveUrl, channel.ProxyUrl, resp.Body) // extract the best quality live url from the master playlist
		if err == nil {
			li := &model.LiveInfo{}
			if !global.IsValidURL(liveUrl) {
				liveUrl = global.GetBaseURL(liveUrl) + liveUrl
			}
			li.LiveUrl = liveUrl
			li.ExtraInfo = prevLiveInfo.ExtraInfo
			return li, nil
		}
	}
	// check if the url is flv or other video streaming format
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "video") {
		log.Println(liveUrl, "is a valid url")
		if err == nil {
			li := &model.LiveInfo{}
			li.LiveUrl = liveUrl
			li.ExtraInfo = prevLiveInfo.ExtraInfo
			return li, nil
		}
	}
	return nil, service.NoMatchFeed
}

func init() {
	service.RegisterPlugin("repeater", &RepeaterParser{}, 6)
}
