package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/snowie2000/livetv/service"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
	"github.com/snowie2000/livetv/playlist/diyp"
	"github.com/snowie2000/livetv/playlist/m3u"
)

type M3UParser struct{}

type ParsedChannel struct {
	ID       int
	Name     string
	Logo     string
	URL      string
	Proxy    bool
	TsProxy  string
	ProxyUrl string
	Category string
}

type M3UPlayList struct {
	Channels []ParsedChannel
}

func (p *M3UParser) Transform(req *http.Request, info *model.LiveInfo) error {
	var ui service.UrlInfo
	json.Unmarshal([]byte(info.ExtraInfo), &ui)
	for v, k := range ui.Headers {
		req.Header.Set(v, k)
	}
	return nil
}

// func (p *M3UParser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
func (p *M3UParser) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	_, err := url.Parse(channel.URL)
	if err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: global.TransportWithProxy(channel.ProxyUrl),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: global.CookieJar,
	}
	req, err := http.NewRequest("GET", channel.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", service.DefaultUserAgent)
	p.Transform(req, &model.LiveInfo{
		ExtraInfo: channel.Extra,
	})
	resp, err := client.Do(req)
	if err != nil {
		return nil, service.RetryOutdated
	}
	defer global.CloseBody(resp)

	if resp.ContentLength > 10*1024*1024 {
		return nil, errors.New("playlist too large")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, service.RetryOutdated
	}

	if playlist, err := m3u.ParseFromReader(bytes.NewBuffer(content)); err == nil {
		parsedList := []ParsedChannel{}
		for i, track := range playlist.Tracks {
			channel := ParsedChannel{
				Category: "",
				ID:       i,
				Name:     track.Name,
				URL:      track.URI,
				Proxy:    false,
				ProxyUrl: channel.ProxyUrl,
				Logo:     "",
			}
			for _, tag := range track.Tags {
				switch tag.Name {
				case "tvg-logo":
					channel.Logo = tag.Value
				case "tvg-name":
					channel.Name = tag.Value
				case "group-title":
					channel.Category = tag.Value
				}
			}
			parsedList = append(parsedList, channel)
		}

		// save parsed channel list into liveinfo
		js, _ := json.Marshal(parsedList)
		li := &model.LiveInfo{}
		li.LiveUrl = ""
		li.ExtraInfo = string(js)
		return li, nil
	}

	// try as DIYP format
	if playlist, err := diyp.ParseChannelFromReader(bytes.NewBuffer(content)); err == nil {
		parsedList := []ParsedChannel{}
		i := 0
		for _, group := range playlist.Groups {
			for _, track := range group.Channels {
				for _, source := range track.Sources {
					channel := ParsedChannel{
						Category: group.Name,
						ID:       i,
						Name:     track.Name,
						URL:      source.Url,
						Proxy:    false,
						ProxyUrl: channel.ProxyUrl,
						Logo:     "",
					}
					parsedList = append(parsedList, channel)
					i++
				}
			}
		}

		// save parsed channel list into liveinfo
		js, _ := json.Marshal(parsedList)
		li := &model.LiveInfo{}
		li.LiveUrl = ""
		li.ExtraInfo = string(js)
		return li, nil
	}
	return nil, errors.New("Unsupported playlist format")
}

// channel provider
func (p *M3UParser) Channels(parentChannel *model.Channel, liveInfo *model.LiveInfo) (channels []*model.Channel) {
	var parsedList []ParsedChannel
	json.Unmarshal([]byte(liveInfo.ExtraInfo), &parsedList)
	for _, it := range parsedList {
		channel := &model.Channel{
			ID:        it.ID,
			ParentID:  parentChannel.ChannelID,
			ChannelID: fmt.Sprintf("%d-%d", parentChannel.ID, it.ID),
			Category:  it.Category,
			Name:      it.Name,
			Logo:      it.Logo,
			Parser:    "auto",
			URL:       it.URL,
			ProxyUrl:  parentChannel.ProxyUrl,
			Proxy:     parentChannel.Proxy,
			TsProxy:   parentChannel.TsProxy,
			Extra:     parentChannel.Extra,
		}
		channels = append(channels, channel)
	}
	return channels
}

func init() {
	service.RegisterPlugin("playlist", &M3UParser{}, 4)
}
