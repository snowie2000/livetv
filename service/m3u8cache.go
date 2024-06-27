package service

import (
	"log"

	"github.com/zjyl1994/livetv/model"

	"github.com/zjyl1994/livetv/global"
)

var updateConcurrent = make(chan bool, 2) // allow up to 2 urls to be updated simultaneously

func LoadChannelCache() {
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return
	}
	for _, v := range channels {
		UpdateURLCacheSingle(v.URL, v.ProxyUrl, v.Parser, true)
	}
}

func UpdateURLCacheSingle(Url string, proxyUrl string, Parser string, bUpdateStatus bool) (*model.LiveInfo, error) {
	updateConcurrent <- true
	defer func() {
		<-updateConcurrent
	}()
	log.Println("caching", Url)
	liveInfo, err := RealLiveM3U8(Url, proxyUrl, Parser)
	if err != nil {
		global.URLCache.Delete(Url)
		UpdateStatus(Url, Error, err.Error())
		log.Println("[LiveTV]", err)
	} else {
		global.URLCache.Store(Url, liveInfo)
		if bUpdateStatus {
			UpdateStatus(Url, Ok, "Live!")
		}
		log.Println(Url, "cached")
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
		UpdateURLCacheSingle(v.URL, v.ProxyUrl, v.Parser, true)
	}
}
