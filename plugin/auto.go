package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
	"github.com/snowie2000/livetv/service"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ParserDetector struct {
	parser service.ChannelParser
}

var (
	errInvalid   = errors.New("Invalid URL")
	pluginMapper = map[string]string{
		"mpegurl": "http",
		"text":    "http",
		"flv":     "streaming",
		"mp4":     "streaming",
	}
	fallbackPlugin = "streaming"
	protocolMapper = map[string]string{
		"rtmp": "rtmp",
	}
)

func (p *ParserDetector) Detect(channel *model.Channel) (string, error) {
	liveUrl := service.CleanUrl(channel.URL)

	// let's scan the protocol first
	{
		u, err := url.Parse(liveUrl)
		if err != nil {
			return "", err
		}
		for p, n := range protocolMapper {
			if strings.EqualFold(p, u.Scheme) {
				log.Println(liveUrl, "=>", n, "by protocol")
				return n, nil
			}
		}
	}

	previousExtraInfo := channel.Extra
	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: global.TransportWithProxy(channel.ProxyUrl),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: global.CookieJar,
	}
	req, err := http.NewRequest("GET", liveUrl, nil)
	if err != nil {
		return "", errInvalid
	}
	req.Header.Set("User-Agent", service.DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer global.CloseBody(resp)
	redir := resp.Header.Get("Location")
	// unpack previousExtraInfo
	var pei service.UrlInfo
	json.Unmarshal([]byte(previousExtraInfo), &pei)
	if redir != "" {
		if pei.RedirectCounter > 5 {
			return "", errors.New("Too many redirections")
		}

		// recreate full url for relative redirections
		if !global.IsValidURL(redir) {
			redir = global.MergeUrl(global.GetBaseURL(liveUrl), redir)
		}

		var ui service.UrlInfo
		decoder := json.NewDecoder(resp.Body)
		if decoder.Decode(&ui) != nil {
			ui = pei
		}

		ui.RedirectCounter = pei.RedirectCounter + 1
		js, _ := json.Marshal(ui)
		nextChannel := *channel
		nextChannel.URL = redir
		nextChannel.Extra = string(js)
		return p.Detect(&nextChannel)
	} else {
		pei.RedirectCounter = 0 // reset counter on a successful parse
	}
	// this is a direct m3u8 url, let's parse the content
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	for n, p := range pluginMapper {
		if strings.Contains(contentType, n) {
			log.Println(liveUrl, "=>", p, "by content-type")
			return p, nil
		}
	}
	log.Println(liveUrl, "=>", p, "as a fallback")
	return fallbackPlugin, nil
}

func (p *ParserDetector) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (info *model.LiveInfo, error error) {
	return nil, fmt.Errorf("This is not a parser")
}

func init() {
	service.RegisterPlugin("auto", &ParserDetector{}, 1)
}
