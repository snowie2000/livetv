// youtube
package plugin

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/snowie2000/livetv/model"

	"github.com/dlclark/regexp2"
)

type YoutubeParser struct{}

type YoutubeExtraInfo struct {
	LastUrl string
}

func isLive(m3u8Url string, proxyUrl string) bool {
	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: transportWithProxy(proxyUrl),
	}
	req, err := http.NewRequest("GET", m3u8Url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	defer resp.Body.Close()
	if resp.ContentLength > 10*1024*1024 || !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "mpegurl") {
		return false
	}
	content, _ := io.ReadAll(resp.Body)
	scontent := string(content)
	return !strings.Contains(scontent, "EXT-X-ENDLIST")
}

func parseUrl(liveUrl string, proxyUrl string) (*model.LiveInfo, error) {
	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: transportWithProxy(proxyUrl),
	}
	req, err := http.NewRequest("GET", liveUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	// DO not parse invalid response, parse HTML only
	if resp.ContentLength > 10*1024*1024 || !strings.Contains(resp.Header.Get("Content-Type"), "html") {
		return nil, errors.New("invalid url")
	}
	content, _ := io.ReadAll(resp.Body)
	scontent := string(content)
	regex := regexp2.MustCompile(`(?<=hlsManifestUrl":").*\.m3u8`, 0)
	matches, _ := regex.FindStringMatch(scontent)
	if matches != nil {
		gps := matches.Groups()
		liveMasterUrl := gps[0].Captures[0].String()
		liveUrl, err := bestFromMasterPlaylist(liveMasterUrl, proxyUrl) // extract the best quality live url from the master playlist
		if err != nil {
			return nil, err
		}

		// check if the live feed is still streaming
		if !isLive(liveUrl, proxyUrl) {
			return nil, errors.New("No longer streaming")
		}

		logo := ""
		logoexp := regexp2.MustCompile(`(?<=owner":{"videoOwnerRenderer":{"thumbnail":{"thumbnails":\[{"url":")[^=]*`, 0)
		logomatches, _ := logoexp.FindStringMatch(scontent)
		if logomatches != nil {
			logo = logomatches.Groups()[0].Captures[0].String()
		}
		var ei YoutubeExtraInfo
		videoexp := regexp2.MustCompile(`"og:url"\s*content="(.+?)"`, 0)
		urlmatches, _ := videoexp.FindStringMatch(scontent)
		if urlmatches != nil {
			ei.LastUrl = urlmatches.Groups()[1].Captures[0].String()
			log.Println("found the real url for video:", ei.LastUrl)
		}
		sExtra, _ := json.Marshal(&ei)

		li := &model.LiveInfo{}
		li.LiveUrl = liveUrl
		li.Logo = logo
		li.ExtraInfo = string(sExtra)
		return li, nil
	}
	return nil, NoMatchFeed
}

func (p *YoutubeParser) Check(content string, info *model.LiveInfo) error {
	if strings.Contains(content, "EXT-X-ENDLIST") {
		return errors.New("live ended")
	}
	return nil
}

func (p *YoutubeParser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
	var info YoutubeExtraInfo
	json.Unmarshal([]byte(previousExtraInfo), &info)
	// for generic urls like "youtube.com/@channel/live", we try last url first, then the generic url
	if getYouTubeVideoID(liveUrl) == "" && info.LastUrl != "" {
		if li, err := parseUrl(info.LastUrl, proxyUrl); err == nil {
			log.Println("Reused last url for video interpretation:", info.LastUrl)
			return li, err
		}
	}
	return parseUrl(liveUrl, proxyUrl)
}

func init() {
	registerPlugin("youtube", &YoutubeParser{})
}
