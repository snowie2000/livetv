package service

import (
	"bufio"
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

func processMediaPlaylist(playlistUrl string, data string, prefixURL string, proxy bool, fnTransform func(raw string, ts string) string) string {
	var sb strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(data))
	baseUrl := global.GetBaseURL(playlistUrl)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "#") {
			sb.WriteString(l)
		} else {
			if !global.IsValidURL(l) {
				l = cleanUrl(global.MergeUrl(baseUrl, l))
			}
			if proxy {
				tsLink := prefixURL + util.CompressString(l)
				if fnTransform != nil {
					tsLink = fnTransform(l, tsLink)
				}
				sb.WriteString(tsLink)
			} else {
				sb.WriteString(l)
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func processMasterPlaylist(playlistUrl string, pl *m3u8.MasterPlaylist, prefixURL string, proxy bool, fnTransform func(raw string, ts string) string) string {
	baseUrl := global.GetBaseURL(playlistUrl)
	handleUri := func(uri string) string {
		if uri == "" {
			return uri
		}
		if !global.IsValidURL(uri) {
			uri = cleanUrl(global.MergeUrl(baseUrl, uri))
		}
		if proxy {
			plLink := prefixURL + util.CompressString(uri)
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
			prefixURL = prefixURL + "/playlist.m3u8?token=" + proxyToken + "&k="
			return processMasterPlaylist(playlistUrl, p.(*m3u8.MasterPlaylist), prefixURL, proxy, fnTransform)
		case m3u8.MEDIA:
			prefixURL = prefixURL + "/live.ts?token=" + proxyToken + "&k="
			return processMediaPlaylist(playlistUrl, data, prefixURL, proxy, fnTransform)
		}
	}
	return ""
}

func init() {
	startUp = time.Now().Unix()
}
