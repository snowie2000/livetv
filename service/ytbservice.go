package service

import (
	"context"
	"errors"
	"github.com/dlclark/regexp2"
	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/syncx"
	"github.com/sosodev/duration"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"log"
	"strings"
)

var (
	errVideoNotFound = errors.New("live stream not found")
	channelIdMap     syncx.Map[string, string]
)

func getYoutubeService() (*youtube.Service, error) {
	apiKey, err := global.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}
	if apiKey == "" {
		return nil, errors.New("API not set")
	}
	ctx := context.Background()
	return youtube.NewService(ctx, option.WithAPIKey(apiKey))
}

func GetChannelIdByName(channel_name string) (string, error) {
	// channel id stays the same for the same channel
	if id, ok := channelIdMap.Load(channel_name); ok {
		return id, nil
	}
	ytb, err := getYoutubeService()
	if err != nil {
		return "", err
	}
	call := ytb.Search.List([]string{"snippet"}).
		Q(channel_name). // Set the search query to the channel name
		Type("channel"). // Restrict search to only channels
		MaxResults(1)
	response, err := call.Do()
	if err != nil {
		return "", err
	}
	if len(response.Items) == 0 || response.Items[0].Id == nil {
		return "", errChannelNotFound
	}
	channelIdMap.Store(channel_name, response.Items[0].Id.ChannelId)
	return response.Items[0].Id.ChannelId, nil
}

func SearchForVideo(channel_id string, keyword string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	ytb, err := getYoutubeService()
	if err != nil {
		return "", err
	}
	call := ytb.Search.List([]string{"snippet"}).
		ChannelId(channel_id).
		Q(keyword).
		Type("video").
		EventType("live").
		MaxResults(10) // we need the best match only
	response, err := call.Do()
	if err != nil {
		return "", err
	}
	for _, video := range response.Items {
		if strings.Contains(video.Id.Kind, "video") && video.Id.VideoId != "" && strings.Contains(video.Snippet.Title, keyword) {
			log.Println("found live:", video.Snippet.Title)
			return "https://www.youtube.com/watch?v=" + video.Id.VideoId, nil
		}
	}
	return "", errVideoNotFound
}

// regex from https://stackoverflow.com/questions/5830387/how-do-i-find-all-youtube-video-ids-in-a-string-using-a-regex?lq=1
func GetYouTubeVideoID(url string) string {
	regex := regexp2.MustCompile(`(?:youtu\.be\/|youtube(?:-nocookie)?\.com\S*?[^\w\s-])([\w-]{11})(?=[^\w-]|$)(?![?=&+%\w.-]*(?:['"][^<>]*>|<\/a>))[?=&+%\w.-]*`, 0)
	match, _ := regex.FindStringMatch(url)
	if match != nil && len(match.Groups()) > 0 {
		return match.Groups()[0].Captures[0].String()
	}
	return ""
}

func GetYouTubeChannelID(url string) string {
	regex := regexp2.MustCompile(`youtu((\.be)|(be\..{2,5}))\/((user)|(channel)|(c)|(@))\/?([a-zA-Z0-9\-_]{1,})`, 0)
	match, _ := regex.FindStringMatch(url)
	if match != nil && len(match.Groups()) > 0 {
		return match.Groups()[9].Captures[0].String()
	}
	return ""
}

/* not possible right now
func GetManifestHLS(vid string) (string, error) {
	service, err := getYoutubeService()
	if err != nil {
		return "", err
	}
	response, err := service.Videos.List([]string{"contentDetails"}).Id(vid).Do()
	if err!=nil {
		return "", err
	}
	if len(response.Items) >0 {
		return response.Items[0].ContentDetails
	}
}
*/

func GetChannelLiveStream(channelName string) (string, error) {
	service, err := getYoutubeService()
	if err != nil {
		return "", err
	}
	response, err := service.Search.List([]string{"snippet"}).ChannelId(channelName).EventType("live").Type("video").Do()
	if err != nil {
		return "", err
	}
	if len(response.Items) > 0 {
		return response.Items[0].Id.VideoId, nil
	}
	return "", errors.New("This channel is not currently live")
}

func GetVideoDuration(url string) (float64, error) {
	vid := GetYouTubeVideoID(url)
	if vid == "" {
		return 0, errors.New("not a valid video url")
	}
	service, err := getYoutubeService()
	if err != nil {
		return 0, err
	}

	// Call the videos.list method with the video ID
	resp, err := service.Videos.List([]string{"contentDetails"}).Id(vid).Do()
	if err == nil && len(resp.Items) > 0 {
		d, err := duration.Parse(resp.Items[0].ContentDetails.Duration)
		if err == nil {
			return d.ToTimeDuration().Seconds(), nil
		}
	}
	return 0, err
}
