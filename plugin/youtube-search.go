// youtube
package plugin

import (
	"errors"
	"github.com/snowie2000/livetv/model"
	"github.com/snowie2000/livetv/service"
	"log"
	"net/url"
)

type YoutubeSearchParser struct {
}

func (p *YoutubeSearchParser) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	u, err := url.Parse(channel.URL)
	if err != nil {
		return nil, err
	}
	parser := u.Query().Get("parser")
	channel_id := u.Query().Get("id")
	channel_name := u.Query().Get("name")
	keyword := u.Query().Get("keyword")
	if (channel_id == "" && channel_name == "") || keyword == "" {
		return nil, errors.New("channel or keyword is empty")
	}
	if parser == "" {
		parser = "youtube" // use built-in youtube parser by default
	}
	Parser, err := service.GetPlugin(parser)
	if err != nil {
		return nil, err
	}
	if channel_id == "" {
		// channel id not provided, search by name
		channel_id, err = service.GetChannelIdByName(channel_name)
		if err != nil {
			return nil, err
		}
		log.Println("channel_id for ", channel_name, "is", channel_id)
	}
	// search for the live-streaming url
	videoUrl, err := service.SearchForVideo(channel_id, keyword)
	if err != nil {
		return nil, err
	}
	log.Println("got video url:", videoUrl)
	// create a temp channel for youtube parser
	ch := *channel
	ch.URL = videoUrl
	return Parser.Parse(&ch, prevLiveInfo)
}

func init() {
	service.RegisterPlugin("ytb-live-search", &YoutubeSearchParser{}, 2)
}
