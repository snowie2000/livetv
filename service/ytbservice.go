package service

import (
	"context"
	"errors"
	"fmt"
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

type LiveVideo struct {
	VideoID string
	Title   string
}

// recentLimit controls how many recent uploads per channel are inspected.
// A value around 20-50 is usually reasonable.
func FindLiveVideos(
	yt *youtube.Service,
	channelID string,
	recentLimit int64,
) ([]LiveVideo, error) {
	if recentLimit <= 0 {
		recentLimit = 20
	}
	if recentLimit > 50 {
		recentLimit = 50
	}

	var result []LiveVideo

	// YouTube Channels.List accepts multiple IDs, up to the API's
	// normal batch limits.
	chResp, err := yt.Channels.
		List([]string{"snippet", "contentDetails"}).
		Id(channelID).
		Do()
	if err != nil {
		return nil, fmt.Errorf("get channels: %w", err)
	}

	for _, ch := range chResp.Items {
		if ch.ContentDetails == nil ||
			ch.ContentDetails.RelatedPlaylists == nil {
			continue
		}

		uploadsPlaylistID := ch.ContentDetails.RelatedPlaylists.Uploads
		if uploadsPlaylistID == "" {
			continue
		}

		// Get recent uploaded video IDs.
		plResp, err := yt.PlaylistItems.
			List([]string{"contentDetails"}).
			PlaylistId(uploadsPlaylistID).
			MaxResults(recentLimit).
			Do()
		if err != nil {
			return nil, fmt.Errorf(
				"get uploads playlist for channel %s: %w",
				ch.Id,
				err,
			)
		}

		videoIDs := make([]string, 0, len(plResp.Items))

		for _, item := range plResp.Items {
			if item.ContentDetails == nil {
				continue
			}

			if item.ContentDetails.VideoId != "" {
				videoIDs = append(
					videoIDs,
					item.ContentDetails.VideoId,
				)
			}
		}

		if len(videoIDs) == 0 {
			continue
		}

		// Fetch authoritative video/live-stream metadata.
		vResp, err := yt.Videos.
			List([]string{
				"snippet",
				"liveStreamingDetails",
			}).
			Id(videoIDs...).
			Do()
		if err != nil {
			return nil, fmt.Errorf(
				"get videos for channel %s: %w",
				ch.Id,
				err,
			)
		}

		for _, video := range vResp.Items {
			if !isCurrentlyLive(video) {
				continue
			}

			live := LiveVideo{
				VideoID: video.Id,
			}

			if ch.Snippet != nil {
				live.Title = ch.Snippet.Title
			}

			if video.Snippet != nil {
				live.Title = video.Snippet.Title
			}
			result = append(result, live)
		}
	}

	return result, nil
}

func isCurrentlyLive(video *youtube.Video) bool {
	if video == nil || video.LiveStreamingDetails == nil {
		return false
	}

	d := video.LiveStreamingDetails

	// Active live broadcast:
	//
	// actualStartTime exists
	// actualEndTime does not exist yet.
	return d.ActualStartTime != "" &&
		d.ActualEndTime == ""
}

func SearchForVideo2(channel_id string, keyword string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	ytb, err := getYoutubeService()
	if err != nil {
		return "", err
	}
	lives, err := FindLiveVideos(ytb, channel_id, 20)
	if err != nil {
		return "", err
	}
	for _, video := range lives {
		log.Println("Checking", video.Title, "...")
		if strings.Contains(video.Title, keyword) {
			log.Println("found live:", video.Title)
			return "https://www.youtube.com/watch?v=" + video.VideoID, nil
		}
	}
	return "", errVideoNotFound
}

func SearchForVideo(channel_id string, keyword string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	keywords := strings.Split(keyword, " ")
	ytb, err := getYoutubeService()
	if err != nil {
		return "", err
	}
	call := ytb.Search.List([]string{"snippet"}).
		ChannelId(channel_id).
		Q(keyword).
		Type("video").
		//EventType("live").
		MaxResults(50) // we need the best match only
	response, err := call.Do()
	if err != nil {
		return "", err
	}
	bestMatch := ""
	for _, video := range response.Items {
		log.Println("Checking video", video.Snippet.Title, "content", video.Snippet.LiveBroadcastContent)
		if strings.Contains(video.Id.Kind, "video") && video.Id.VideoId != "" {
			found := true
			for _, k := range keywords {
				if !strings.Contains(video.Snippet.Title, k) {
					found = false
					break
				}
			}
			if found {
				log.Println("found live:", video.Snippet.Title)
				if video.Snippet.LiveBroadcastContent == "live" {
					return "https://www.youtube.com/watch?v=" + video.Id.VideoId, nil
				}
				if bestMatch == "" {
					bestMatch = "https://www.youtube.com/watch?v=" + video.Id.VideoId
				}
			}
		}
	}
	if bestMatch != "" {
		return bestMatch, nil
	}
	return "", errVideoNotFound
}

// regex from https://stackoverflow.com/questions/5830387/how-do-i-find-all-youtube-video-ids-in-a-string-using-a-regex?lq=1
func GetYouTubeVideoID(url string) string {
	regex := regexp2.MustCompile(`(?:youtu\.be\/|youtube(?:-nocookie)?\.com\S*?[^\w\s-])([\w-]{11})(?=[^\w-]|$)(?![?=&+%\w.-]*(?:['"][^<>]*>|<\/a>))[?=&+%\w.-]*`, 0)
	match, _ := regex.FindStringMatch(url)
	if match != nil && len(match.Groups()) > 1 {
		return match.Groups()[1].Captures[0].String()
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
