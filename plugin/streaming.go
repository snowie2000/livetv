package plugin

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
	"github.com/snowie2000/livetv/service"
	"github.com/snowie2000/livetv/util"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type StreamingParser struct {
	URLM3U8Parser
}

func (p *StreamingParser) Host(c *gin.Context, info *model.LiveInfo, chInfo *model.Channel) error {
	proxyTarget := info.LiveUrl
	if chInfo.Proxy {
		// if proxy stream is enabled, redirect to the universal reverse proxy with our secret
		proxyTarget = fmt.Sprintf("/proxy?token=%s&k=%s&proxy=%s", global.GetLiveToken(), util.CompressString(info.LiveUrl), url.QueryEscape(chInfo.ProxyUrl))
		if chInfo.TsProxy == "" {
			proxyTarget = path.Join(chInfo.TsProxy, proxyTarget)
		}
	}
	http.Redirect(c.Writer, c.Request, proxyTarget, http.StatusFound)
	return nil
}

func (p *StreamingParser) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	u, err := url.Parse(channel.URL)
	if err == nil && strings.HasPrefix(strings.ToLower(u.Scheme), "http") {
		li := &model.LiveInfo{}
		li.LiveUrl = u.String()
		li.ExtraInfo = prevLiveInfo.ExtraInfo
		return li, nil
	}
	return nil, service.NoMatchFeed
}

func init() {
	service.RegisterPlugin("streaming", &StreamingParser{}, 3)
}
