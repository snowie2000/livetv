package service

import (
	"log"

	"github.com/LgoLgo/geentrant"
	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
	"github.com/snowie2000/livetv/plugin"
)

var updateConcurrent = &greetrant.RecursiveMutex{}

func LoadChannelCache() {
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return
	}
	for _, v := range channels {
		UpdateURLCacheSingle(v, true)
	}
	InvalidateChannelCache()
}

func UpdateSubChannels(parentChannel *model.Channel, liveInfo *model.LiveInfo, Parser string, bUpdateStatus bool) {
	// let's check if there are any sub channels
	if p, err := plugin.GetPlugin(Parser); err == nil {
		if provider, ok := p.(plugin.ChannalProvider); ok {
			subchannels := provider.Channels(parentChannel, liveInfo)
			for _, ch := range subchannels {
				UpdateURLCacheSingle(ch, bUpdateStatus)
			}
		}
	}
}

func UpdateURLCacheSingle(channel *model.Channel, bUpdateStatus bool) (*model.LiveInfo, error) {
	updateConcurrent.Lock()
	defer func() {
		updateConcurrent.Unlock()
	}()
	log.Println("caching", channel.URL)
	liveInfo, err := RealLiveM3U8(channel.URL, channel.ProxyUrl, channel.Parser)
	if err != nil {
		global.URLCache.Delete(channel.URL)
		UpdateStatus(channel.URL, Error, err.Error())
		log.Println("[LiveTV]", err)
	} else {
		// cache parsed result
		global.URLCache.Store(channel.URL, liveInfo)
		if bUpdateStatus {
			UpdateStatus(channel.URL, Ok, "Live!")
		}
		log.Println(channel.URL, "cached")

		InvalidateChannelCache(channel.ChannelID)
		UpdateSubChannels(channel, liveInfo, channel.Parser, bUpdateStatus)
	}
	return liveInfo, err
}

func UpdateURLCache() {
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return
	}
	urlcache := make(map[string]bool)
	for _, v := range channels {
		urlcache[v.URL] = true
	}
	// delete urlcaches that we do not serve anymore
	global.URLCache.Range(func(k string, info *model.LiveInfo) bool {
		if _, ok := urlcache[k]; !ok {
			global.URLCache.Delete(k)
			DeleteStatus(k)
			return true
		}
		return true
	})
	for _, v := range channels {
		UpdateURLCacheSingle(v, true)
	}
	InvalidateChannelCache()
}
