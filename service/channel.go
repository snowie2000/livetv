package service

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"

	"github.com/zjyl1994/livetv/global"
	"github.com/zjyl1994/livetv/model"
)

func GetAllChannel() (channels []model.Channel, err error) {
	err = global.DB.Find(&channels).Error
	if err == nil {
		// update all channel info to the cache
		for i := range channels {
			if ch, ok := global.ChannelCache.Load(channels[i].ID); ok {
				channels[i].Token = ch.Token
			} else {
				channels[i].Token = generateToken(channels[i].ID)
				global.ChannelCache.Store(channels[i].ID, channels[i])
			}
		}
	}
	return
}

func SaveChannel(channel model.Channel) error {
	global.ChannelCache.Delete(channel.ID)
	return global.DB.Save(&channel).Error
}

func DeleteChannel(id uint) error {
	global.ChannelCache.Delete(id)
	return global.DB.Delete(model.Channel{}, "id = ?", id).Error
}

func GetChannel(channelNumber uint) (channel model.Channel, err error) {
	if ch, ok := global.ChannelCache.Load(channelNumber); ok {
		return ch, nil
	}
	err = global.DB.Where("id = ?", channelNumber).First(&channel, channelNumber).Error
	if err == nil {
		channel.Token = generateToken(channelNumber)
		global.ChannelCache.Store(channelNumber, channel)
	}
	return
}

const SALT string = "LiVeTv"

func generateToken(channelNumber uint) string {
	secret := global.GetSecretToken()
	if secret == "" {
		return ""
	}
	text := fmt.Sprintf("%s_%s_%d", secret, SALT, channelNumber)
	hash := md5.Sum([]byte(text))
	return base64.URLEncoding.EncodeToString(hash[:])[1:10]
}
