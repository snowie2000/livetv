package plugin

import (
	"github.com/snowie2000/livetv/model"
)

type M3URepeater struct {
	M3UParser
}

// channel provider
func (p *M3URepeater) Channels(parentChannel *model.Channel, liveInfo *model.LiveInfo) (channels []*model.Channel) {
	channels = p.M3UParser.Channels(parentChannel, liveInfo)
	for _, it := range channels {
		it.Parser = "repeater"
	}
	return channels
}

func init() {
	registerPlugin("playlist-repeater", &M3URepeater{}, 5)
}
