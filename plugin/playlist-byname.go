package plugin

import (
	"encoding/json"
	"fmt"
	"github.com/snowie2000/livetv/model"
	"github.com/snowie2000/livetv/service"
	"github.com/snowie2000/livetv/syncx"
	"net/url"
)

var (
	channelIndex syncx.Map[string, *syncx.HashedSlice[*model.Channel]]
)

type M3USearcher struct {
	M3UParser
}

func (p *M3USearcher) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	li, err := p.M3UParser.Parse(channel, prevLiveInfo)
	if err == nil {
		var parsedList []ParsedChannel
		json.Unmarshal([]byte(li.ExtraInfo), &parsedList)
		chMap := syncx.NewHashedSlice[*model.Channel]()
		for _, it := range parsedList {
			channel := &model.Channel{
				ID:        it.ID,
				ParentID:  channel.ChannelID,
				ChannelID: fmt.Sprintf("%d-%d", channel.ID, it.ID),
				Category:  it.Category,
				Name:      it.Name,
				Logo:      it.Logo,
				Parser:    "http",
				URL:       it.URL,
				ProxyUrl:  channel.ProxyUrl,
				Proxy:     channel.Proxy,
				TsProxy:   channel.TsProxy,
				Extra:     channel.Extra,
			}
			channel.CustomQueryString = fmt.Sprintf("sid=%s", channel.Digest())
			chMap.Add(channel)
		}
		channelIndex.Store(channel.ChannelID, chMap)
	}
	return li, err
}

func (p *M3USearcher) ParseChannelUrl(chUrl string, mainChannelInfo *model.Channel) *model.Channel {
	u, err := url.Parse(chUrl)
	if err != nil {
		return nil
	}
	searchId := u.Query().Get("sid")
	if channelMap, ok := channelIndex.Load(mainChannelInfo.ChannelID); ok {
		if ch, ok := channelMap.GetByDigest(searchId); ok {
			return ch
		}
	}
	return nil
}

// channel provider
func (p *M3USearcher) Channels(parentChannel *model.Channel, liveInfo *model.LiveInfo) (channels []*model.Channel) {
	if channelMap, ok := channelIndex.Load(parentChannel.ChannelID); ok {
		return channelMap.AsSlice()
	}
	return nil
}

func init() {
	service.RegisterPlugin("playlist-byname", &M3USearcher{}, 5)
}
