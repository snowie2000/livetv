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
		idx := 0
		for _, it := range parsedList {
			subCh := &model.Channel{
				ID:        idx,
				ParentID:  channel.ChannelID,
				ChannelID: fmt.Sprintf("%d-%d", channel.ID, idx),
				Category:  it.Category,
				Name:      it.Name,
				Logo:      it.Logo,
				Parser:    "auto",
				URL:       it.URL,
				ProxyUrl:  channel.ProxyUrl,
				Proxy:     channel.Proxy,
				TsProxy:   channel.TsProxy,
				Extra:     channel.Extra,
			}
			subCh.CustomQueryString = fmt.Sprintf("sid=%s", subCh.Digest())
			if chMap.Add(subCh) == nil { // ignore duplicate channels
				idx++
			}
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
