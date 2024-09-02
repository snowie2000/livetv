// youtube
package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os/exec"
	"strings"

	"github.com/snowie2000/livetv/model"

	"github.com/snowie2000/livetv/global"
)

type YtDlpOAuthParser struct{}

type YtVideoAudioInfo struct {
	VideoUrl string
	AudioUrl string
}

var m3u8Template string = `
#EXTM3U
#EXT-X-VERSION:4
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="audio_mp4a.40.2_48000",NAME="a48000_zho",DEFAULT=YES,AUTOSELECT=YES,URI="{AUDIO}"
#EXT-X-STREAM-INF:PROGRAM-ID=0,BANDWIDTH=2152910,AUDIO="audio_mp4a.40.2_48000"
{VIDEO}
`

func (p *YtDlpOAuthParser) ForgeM3U8(info *model.LiveInfo) (baseUrl string, body string, err error) {
	var videoInfo YtVideoAudioInfo
	json.Unmarshal([]byte(info.ExtraInfo), &videoInfo)
	if videoInfo.AudioUrl != "" && videoInfo.VideoUrl != "" {
		pl := m3u8Template
		pl = strings.Replace(pl, "{AUDIO}", videoInfo.AudioUrl, 1)
		pl = strings.Replace(pl, "{VIDEO}", videoInfo.VideoUrl, 1)
		return info.LiveUrl, pl, nil
	}
	return "", "", errors.New("invalid youtube video extracted")
}

func (p *YtDlpOAuthParser) Parse(liveUrl string, proxyUrl string, lastInfo string) (*model.LiveInfo, error) {
	YtdlCmd, err := global.GetConfig("ytdl_cmd")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	YtdlArgs := "--no-warnings --extractor-args youtube:skip=dash -f bestvideo*+bestaudio/best -g --username oauth2 --password '' {url}"
	ytdlArgs := strings.Fields(YtdlArgs)
	for i, v := range ytdlArgs {
		if strings.EqualFold(v, "{url}") {
			ytdlArgs[i] = liveUrl
		}
	}
	_, err = exec.LookPath(YtdlCmd)
	if err != nil {
		log.Println(err)
		return nil, err
	} else {
		ctx, cancelFunc := context.WithTimeout(context.Background(), global.HttpClientTimeout)
		defer cancelFunc()
		cmd := exec.CommandContext(ctx, YtdlCmd, ytdlArgs...)
		out, err := cmd.CombinedOutput()
		output := strings.TrimSpace(string(out))
		lines := strings.Split(output, "\n")
		// this should give two lines: the first is the video m3u8 and the second is the audio m3u8
		if err == nil && len(lines) == 2 {
			videoInfo := &YtVideoAudioInfo{lines[0], lines[1]}
			js, _ := json.Marshal(videoInfo)
			li := &model.LiveInfo{}
			li.ExtraInfo = string(js)
			li.LiveUrl = videoInfo.VideoUrl
			return li, err
		} else {
			if output == "" {
				return nil, err
			} else {
				return nil, errors.Join(errors.New(output+" , "), err)
			}
		}
	}
}

func init() {
	registerPlugin("yt-dlp-oauth", &YtDlpOAuthParser{})
}
