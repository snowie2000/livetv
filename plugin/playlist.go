package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func (p *M3UParser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
	_, err := url.Parse(liveUrl)
	if err != nil {
		return nil, err
	}

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
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer global.CloseBody(resp)

	if resp.ContentLength > 10*1024*1024 {
		return nil, errors.New("playlist too large")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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
				ProxyUrl: proxyUrl,
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
						ProxyUrl: proxyUrl,
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
			ChannelID: fmt.Sprintf("%d-%d", parentChannel.ID, it.ID),
			Category:  it.Category,
			Name:      it.Name,
			Logo:      it.Logo,
			Parser:    "http",
			URL:       it.URL,
			ProxyUrl:  parentChannel.ProxyUrl,
			Proxy:     parentChannel.Proxy,
			TsProxy:   parentChannel.TsProxy,
		}
		channels = append(channels, channel)
	}
	return channels
}

func init() {
	registerPlugin("playlist", &M3UParser{}, 4)
}
