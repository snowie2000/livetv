package service

import (
	"bytes"
	"fmt"
	"github.com/grafov/m3u8"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/util"
)

var startUp int64 = 0

func PlaceHolderHLS() string {
	// t := (time.Now().Unix() - startUp) / 60
	baseUrl, _ := global.GetConfig("base_url")
	if !strings.HasSuffix(baseUrl, "/") {
		baseUrl += "/"
	}
	placeholder := baseUrl + "placeholder.ts"
	tpl := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-MEDIA-SEQUENCE:1
#EXT-X-TARGETDURATION:30
#EXT-X-DISCONTINUITY:0
#EXTINF:30.000000,
%s?t=1
#EXTINF:30.000000,
%s?t=2
#EXTINF:30.000000,
%s?t=3
#EXT-X-ENDLIST
`
	return fmt.Sprintf(tpl, placeholder, placeholder, placeholder)
}

func cleanUrl(Url string) string {
	parsedURL, err := url.Parse(Url)
	if err != nil {
		return Url
	}

	// Resolve the path using path resolution
	parsedURL.Path = path.Clean(parsedURL.Path) // Remove trailing segments

	// Get the final clean URL as a string
	cleanURL := parsedURL.String()

	return cleanURL
}

func processMediaPlaylist(playlistUrl string, pl *m3u8.MediaPlaylist, prefixURL string, proxyToken string, proxy bool, fnTransform func(raw string, ts string) string) string {
	baseUrl := global.GetBaseURL(playlistUrl)
	handleUri := func(uri string) string {
		if uri == "" {
			return uri
		}
		if !global.IsValidURL(uri) {
			uri = cleanUrl(global.MergeUrl(baseUrl, uri))
		}
		if proxy {
			tsLink := global.MergeUrl(prefixURL, "/live.ts?token="+proxyToken+"&k="+util.CompressString(uri))
			if fnTransform != nil {
				tsLink = fnTransform(uri, tsLink)
			}
			uri = tsLink
		}
		return uri
	}

	var i uint
	for i = pl.Count() - pl.WinSize(); i < pl.Count(); i++ {
		pl.Segments[i].URI = handleUri(pl.Segments[i].URI)
	}
	// remove unused segments
	for pl.Count() > pl.WinSize() {
		pl.Remove()
	}
	return pl.Encode().String()
}

func processMasterPlaylist(playlistUrl string, pl *m3u8.MasterPlaylist, prefixURL string, proxyToken string, proxy bool, fnTransform func(raw string, ts string) string) string {
	baseUrl := global.GetBaseURL(playlistUrl)
	handleUri := func(uri string) string {
		if uri == "" {
			return uri
		}
		if !global.IsValidURL(uri) {
			uri = cleanUrl(global.MergeUrl(baseUrl, uri))
		}
		if proxy {
			plLink := global.MergeUrl(prefixURL, "/playlist.m3u8?token="+proxyToken+"&k="+util.CompressString(uri))
			if fnTransform != nil {
				plLink = fnTransform(uri, plLink)
			}
			uri = plLink
		}
		return uri
	}

	for _, v := range pl.Variants {
		v.URI = handleUri(v.URI)
		if v.VariantParams.Alternatives != nil && len(v.VariantParams.Alternatives) > 0 {
			for _, alter := range v.VariantParams.Alternatives {
				alter.URI = handleUri(alter.URI)
			}
		}
	}
	return pl.Encode().String()
}

func M3U8Process(playlistUrl string, data string, prefixURL string, proxyToken string, proxy bool, fnTransform func(raw string, ts string) string) string {
	p, listType, err := m3u8.DecodeFrom(bytes.NewBufferString(data), true)
	if err == nil {
		switch listType {
		case m3u8.MASTER:
			return processMasterPlaylist(playlistUrl, p.(*m3u8.MasterPlaylist), prefixURL, proxyToken, proxy, fnTransform)
		case m3u8.MEDIA:
			return processMediaPlaylist(playlistUrl, p.(*m3u8.MediaPlaylist), prefixURL, proxyToken, proxy, fnTransform)
		}
	}
	return ""
}

func init() {
	startUp = time.Now().Unix()
}
