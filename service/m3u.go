package service

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"
)

func M3UGenerate() (string, error) {
	baseUrl, err := global.GetConfig("base_url")
	if err != nil {
		log.Println(err)
		return "", err
	}
	channels, err := GetAllChannel()
	if err != nil {
		log.Println(err)
		return "", err
	}
	var m3u strings.Builder
	writeChannel := func(ch *model.Channel) {
		logo := ""
		category := "LiveTV"
		if ch.Category != "" {
			category = ch.Category
		}
		if info, ok := global.URLCache.Load(ch.URL); ok {
			logo = info.Logo
		}
		if ch.Logo != "" {
			logo = ch.Logo
		}
		liveData := fmt.Sprintf("#EXTINF:-1, tvg-name=%s tvg-logo=%s group-title=%s, %s\n", strconv.Quote(ch.Name), strconv.Quote(logo), strconv.Quote(category), ch.Name)
		m3u.WriteString(liveData)
		m3u.WriteString(fmt.Sprintf("%s/live.m3u8?token=%s&c=%s\n", baseUrl, ch.Token, ch.ChannelID))
	}
	m3u.WriteString("#EXTM3U\n")
	for _, v := range channels {
		if len(v.Children) > 0 {
			for _, sub := range v.Children {
				writeChannel(sub)
			}
		} else {
			writeChannel(v)
		}
	}
	return m3u.String(), nil
}
