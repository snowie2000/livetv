// youtube
package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/snowie2000/livetv/service"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"

	"github.com/dlclark/regexp2"
)

type Client struct {
	ClientName       string `json:"clientName"`
	ClientVersion    string `json:"clientVersion"`
	UserAgent        string `json:"userAgent"`
	OsName           string `json:"osName"`
	OsVersion        string `json:"osVersion"`
	Hl               string `json:"hl"`
	TimeZone         string `json:"timeZone"`
	UtcOffsetMinutes int    `json:"utcOffsetMinutes"`
}

type Context struct {
	Client Client `json:"client"`
}

type ContentPlaybackContext struct {
	Html5Preference    string `json:"html5Preference"`
	SignatureTimestamp int    `json:"signatureTimestamp"`
}

type PlaybackContext struct {
	ContentPlaybackContext ContentPlaybackContext `json:"contentPlaybackContext"`
}

type VideoRequest struct {
	Context         Context         `json:"context"`
	VideoID         string          `json:"videoId"`
	PlaybackContext PlaybackContext `json:"playbackContext"`
	ContentCheckOk  bool            `json:"contentCheckOk"`
	RacyCheckOk     bool            `json:"racyCheckOk"`
}

type StreamingData struct {
	ExpiresInSeconds      string `json:"expiresInSeconds"`
	AdaptiveFormats       any    `json:"adaptiveFormats"`
	DashManifestUrl       string `json:"dashManifestUrl"`
	HlsManifestUrl        string `json:"hlsManifestUrl"`
	ServerAbrStreamingUrl string `json:"serverAbrStreamingUrl"`
}

type VideoResponse struct {
	ResponseContext   any           `json:"responseContext"`
	PlayabilityStatus any           `json:"playabilityStatus"`
	StreamingData     StreamingData `json:"streamingData"`
}

const userAgent = "com.google.android.youtube/20.10.38 (Linux; U; Android 11) gzip"

type YoutubeParser struct{}

type YoutubeExtraInfo struct {
	LastUrl string
}

func isLive(m3u8Url string, proxyUrl string) bool {
	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: global.TransportWithProxy(proxyUrl),
		Jar:       global.CookieJar,
	}
	req, err := http.NewRequest("GET", m3u8Url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", service.DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return false
	}

	defer global.CloseBody(resp)
	if resp.ContentLength > 10*1024*1024 || !strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "mpegurl") {
		return false
	}
	content, _ := io.ReadAll(resp.Body)
	scontent := string(content)
	return !strings.Contains(scontent, "EXT-X-ENDLIST")
}

func getVideoIdFromHtml(html string) string {
	reg := regexp.MustCompile(`<meta\s+itemprop="identifier"\s+content="(.+?)"`)
	match := reg.FindStringSubmatch(html)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func makeRequestBody(videoId string) *VideoRequest {
	return &VideoRequest{
		Context: Context{
			Client: Client{
				ClientName:       "ANDROID",
				ClientVersion:    "20.10.38",
				UserAgent:        userAgent,
				OsName:           "Android",
				OsVersion:        "11",
				Hl:               "en",
				TimeZone:         "UTC",
				UtcOffsetMinutes: 0,
			},
		},
		VideoID: videoId,
		PlaybackContext: PlaybackContext{
			ContentPlaybackContext: ContentPlaybackContext{
				Html5Preference:    "HTML5_PREF_WANTS",
				SignatureTimestamp: 20458,
			},
		},
		ContentCheckOk: true,
		RacyCheckOk:    true,
	}
}

func parseUrl(liveUrl string, proxyUrl string) (*model.LiveInfo, error) {
	client := http.Client{
		Timeout:   time.Second * 10,
		Transport: global.TransportWithProxy(proxyUrl),
		Jar:       global.CookieJar,
	}
	req, err := http.NewRequest("GET", liveUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", service.DefaultUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer global.CloseBody(resp)
	// DO not parse invalid response, parse HTML only
	if resp.ContentLength > 10*1024*1024 || !strings.Contains(resp.Header.Get("Content-Type"), "html") {
		return nil, errors.New("invalid url")
	}
	content, _ := io.ReadAll(resp.Body)
	scontent := string(content)
	videoId := getVideoIdFromHtml(scontent) // extract video id from the html metadata
	bodyJson := makeRequestBody(videoId)
	body := &bytes.Buffer{}
	encoder := json.NewEncoder(body)
	encoder.Encode(bodyJson)
	req, _ = http.NewRequest("POST", "https://www.youtube.com/youtubei/v1/player?prettyPrint=false", body)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Youtube-Client-Name", "3")
	req.Header.Set("X-Youtube-Client-Version", "20.10.38")
	jsonResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer global.CloseBody(jsonResp)
	var videoResp VideoResponse
	decoder := json.NewDecoder(jsonResp.Body)
	err = decoder.Decode(&videoResp)
	if err != nil {
		return nil, err
	}

	liveMasterUrl := videoResp.StreamingData.HlsManifestUrl
	liveUrl, err = service.BestFromMasterPlaylist(liveMasterUrl, proxyUrl) // extract the best quality live url from the master playlist
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

func (p *YoutubeParser) Check(content string, info *model.LiveInfo) error {
	if strings.Contains(content, "EXT-X-ENDLIST") {
		return errors.New("live ended")
	}
	return nil
}

func (p *YoutubeParser) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	previousExtraInfo := prevLiveInfo.ExtraInfo
	var info YoutubeExtraInfo
	json.Unmarshal([]byte(previousExtraInfo), &info)
	// for generic urls like "youtube.com/@channel/live", we try last url first, then the generic url
	if service.GetYouTubeVideoID(channel.URL) == "" && info.LastUrl != "" {
		if li, err := parseUrl(info.LastUrl, channel.ProxyUrl); err == nil {
			log.Println("Reused last url for video interpretation:", info.LastUrl)
			return li, err
		}
	}
	return parseUrl(channel.URL, channel.ProxyUrl)
}

func init() {
	service.RegisterPlugin("youtube", &YoutubeParser{}, 2)
}
