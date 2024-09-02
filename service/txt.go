package service

import (
	"fmt"
	"log"
	"strings"

	"github.com/snowie2000/livetv/global"
)

type genre struct {
	name     string
	channels map[string][]string
	list     []string
}

func (g *genre) addChannel(chName string, url string) {
	chName = strings.Replace(chName, ",", "_", -1)
	if group, ok := g.channels[chName]; ok {
		group = append(group, url)
		g.channels[chName] = group
	} else {
		g.channels[chName] = []string{url}
		g.list = append(g.list, chName)
	}
}

func (g *genre) String() string {
	channels := []string{g.name + ",#genre#"}
	for _, group := range g.list {
		for _, url := range g.channels[group] {
			channels = append(channels, group+","+url)
		}
	}
	return strings.Join(channels, "\n")
}

func TXTGenerate() (string, error) {
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
	genres := make(map[string]*genre)
	var genreList []string
	for _, v := range channels {
		category := "LiveTV"
		if v.Category != "" {
			category = v.Category
		}
		if g, ok := genres[category]; ok {
			g.addChannel(v.Name, fmt.Sprintf("%s/live.m3u8?token=%s&c=%d", baseUrl, v.Token, v.ID))
		} else {
			g = &genre{
				name:     category,
				channels: make(map[string][]string),
			}
			genreList = append(genreList, category)
			g.addChannel(v.Name, fmt.Sprintf("%s/live.m3u8?token=%s&c=%d", baseUrl, v.Token, v.ID))
			genres[category] = g
		}
	}
	var txt strings.Builder
	for _, category := range genreList {
		txt.WriteString(genres[category].String())
		txt.WriteString("\n\n")
	}
	return txt.String(), nil
}
