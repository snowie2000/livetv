package service

import (
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"

	"github.com/snowie2000/livetv/plugin"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
)

var (
	errChannelNotFound = errors.New("Channel not found")
)

// load channel and its sub channels into cache, generate everything necessary
func loadChannel(ch *model.Channel) {
	cache, ok := global.ChannelCache.Load(strconv.Itoa(ch.ID))
	if ok {
		*ch = cache
	} else {
		ch.ChannelID = strconv.Itoa(ch.ID) // generate string ID for channel structs
		ch.Token = generateToken(ch.ChannelID)
		if ch.HasSubChannel {
			// get child channels
			if liveInfo, ok := global.URLCache.Load(ch.URL); ok {
				if p, err := plugin.GetPlugin(ch.Parser); err == nil {
					if provider, ok := p.(plugin.ChannalProvider); ok {
						ch.Children = provider.Channels(ch, liveInfo)
					}
				}
			}
		}
		if len(ch.Children) > 0 {
			for _, sub := range ch.Children {
				cache, ok := global.ChannelCache.Load(sub.ChannelID)
				if ok {
					sub.Token = cache.Token
				} else {
					sub.Token = generateToken(sub.ChannelID)
					global.ChannelCache.Store(sub.ChannelID, *sub)
				}
			}
		}
		global.ChannelCache.Store(strconv.Itoa(ch.ID), *ch)
	}
}

func GetAllChannel() (channels []*model.Channel, err error) {
	err = global.DB.Find(&channels).Error
	if err == nil {
		// update all channel info to the cache
		for _, ch := range channels {
			loadChannel(ch)
		}
	}
	return
}

func SaveChannel(channel *model.Channel) error {
	global.ChannelCache.Delete(channel.ChannelID)
	// clear children info before saving
	children := channel.Children
	channel.Children = []*model.Channel{}
	err := global.DB.Save(channel).Error
	channel.Children = children
	return err
}

func DeleteChannel(id int) error {
	global.ChannelCache.Delete(strconv.Itoa(id))
	return global.DB.Delete(model.Channel{}, "id = ?", id).Error
}

func InvalidateChannelCache(channels ...string) {
	if len(channels) > 0 {
		for _, ch := range channels {
			global.ChannelCache.Delete(ch)
		}
	} else {
		global.ChannelCache.Clear()
	}
}

func GetChannel(channelNumber int, subNumber int) (*model.Channel, error) {
	chId := strconv.Itoa(channelNumber)
	if subNumber >= 0 {
		chId += "-" + strconv.Itoa(subNumber)
	}

	if ch, ok := global.ChannelCache.Load(chId); ok {
		return &ch, nil
	}

	if subNumber >= 0 {
		// main channel has been loaded, but sub channel can't be found -> invalid channel
		if _, ok := global.ChannelCache.Load(strconv.Itoa(channelNumber)); ok {
			return nil, errChannelNotFound
		}
	}

	// load main channel from db
	var channel model.Channel
	err := global.DB.Where("id = ?", channelNumber).First(&channel, channelNumber).Error
	if err == nil {
		loadChannel(&channel)
	}
	// now load from cache again
	if ch, ok := global.ChannelCache.Load(chId); ok {
		return &ch, nil
	} else {
		return nil, errChannelNotFound
	}
}

const SALT string = "LiVeTv"

func generateToken(channelNumber string) string {
	secret := global.GetSecretToken()
	if secret == "" {
		return ""
	}
	text := fmt.Sprintf("%s_%s_%s", secret, SALT, channelNumber)
	hash := md5.Sum([]byte(text))
	return base64.URLEncoding.EncodeToString(hash[:])[1:10]
}
