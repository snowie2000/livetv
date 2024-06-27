package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/sosodev/duration"
	"github.com/zjyl1994/livetv/global"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
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

// regex from https://stackoverflow.com/questions/5830387/how-do-i-find-all-youtube-video-ids-in-a-string-using-a-regex?lq=1
func GetYouTubeVideoID(url string) string {
	regex := regexp.MustCompile(`(?:youtu\.be\/|youtube(?:-nocookie)?\.com\S*?[^\w\s-])([\w-]{11})(?=[^\w-]|$)(?![?=&+%\w.-]*(?:['"][^<>]*>|<\/a>))[?=&+%\w.-]*`)
	match := regex.FindStringSubmatch(url)
	if match != nil && len(match) > 0 {
		return match[1]
	}
	return ""
}

func GetYouTubeChannelID(url string) string {
	regex := regexp.MustCompile(`youtu((\.be)|(be\..{2,5}))\/((user)|(channel)|(c)|(@))\/?([a-zA-Z0-9\-_]{1,})`)
	match := regex.FindStringSubmatch(url)
	if match != nil && len(match) > 9 {
		return match[9]
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
