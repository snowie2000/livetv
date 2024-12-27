package global

import (
	"time"

	"github.com/snowie2000/livetv/model"

	"github.com/patrickmn/go-cache"
	"github.com/snowie2000/livetv/syncx"
)

var defaultConfigValue = map[string]string{
	"ytdl_cmd":  "yt-dlp",
	"ytdl_args": "--extractor-args youtube:skip=dash -f b -g {url}",
	"base_url":  "http://127.0.0.1:9000",
	"password":  "password",
	"apiKey":    "",
}

var (
	HttpClientTimeout = 10 * time.Second
	ConfigCache       syncx.Map[string, string]
	URLCache          syncx.Map[string, *model.LiveInfo]
	ChannelCache      syncx.Map[string, model.Channel]
	M3U8Cache         = cache.New(3*time.Second, 10*time.Second)
)
