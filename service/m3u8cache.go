package service

import (
	"errors"
	"log"
	"sync"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
	"golang.org/x/sync/singleflight"
)

var (
	cacheGroup singleflight.Group
	cancelers  sync.Map
)

// cancel the parser if the channel or its sub channel is being parsed.
func CancelChannelParser(chId int) {
	if canceler, ok := cancelers.Load(chId); ok {
		if fn, ok := canceler.(func()); ok {
			fn()
		}
	}
}

func runConcurrentUpdates(channels []*model.Channel) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for _, v := range channels {
		wg.Add(1)
		go func(ch *model.Channel) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			UpdateURLCacheSingle(ch, true)
		}(v)
	}
	wg.Wait()
}

func LoadChannelCache() {
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return
	}
	runConcurrentUpdates(channels)
	InvalidateChannelCache()
}

func UpdateSubChannels(parentChannel *model.Channel, liveInfo *model.LiveInfo, Parser string, bUpdateStatus bool) {
	// let's check if there are any sub channels
	if p, err := GetPlugin(Parser); err == nil {
		if provider, ok := p.(ChannalProvider); ok {
			subchannels := provider.Channels(parentChannel, liveInfo)
			if len(subchannels) > 0 {
				canceled := false
				cancelers.Store(parentChannel.ID, func() {
					canceled = true
				})
				defer cancelers.Delete(parentChannel.ID)

				for _, ch := range subchannels {
					if canceled {
						log.Println("Channel has been removed")
						break
					}
					UpdateURLCacheSingle(ch, bUpdateStatus)
				}
			}
		}
	}
}

func UpdateURLCacheSingle(channel *model.Channel, bUpdateStatus bool) (*model.LiveInfo, error) {
	res, err, _ := cacheGroup.Do(channel.URL, func() (any, error) {
		log.Println("caching", channel.URL)
		liveInfo, err := RealLiveM3U8(channel)
		if err != nil {
			if errors.Is(err, RetryOutdated) {
				UpdateStatus(channel.URL, Warning, err.Error())
			} else {
				global.URLCache.Delete(channel.URL)
				UpdateStatus(channel.URL, Error, err.Error())
			}
			log.Println("[LiveTV]", err)
		} else {
			// cache parsed result
			global.URLCache.Store(channel.URL, liveInfo)
			if bUpdateStatus {
				UpdateStatus(channel.URL, Ok, "Live!")
			}
			log.Println(channel.URL, "cached")

			UpdateSubChannels(channel, liveInfo, channel.Parser, bUpdateStatus)
			if channel.ChannelID == channel.ParentID { // invalidate and reload primary channel only (sub channels will be invalidated automatically)
				InvalidateChannelCache(channel)
			}
		}
		return liveInfo, err
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.(*model.LiveInfo), nil
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
	runConcurrentUpdates(channels)
	InvalidateChannelCache()
}
