// youtube
package plugin

import (
	"context"
	"errors"
	"github.com/snowie2000/livetv/service"
	"log"
	"os/exec"
	"strings"

	"github.com/snowie2000/livetv/model"

	"github.com/snowie2000/livetv/global"
)

type YtDlpParser struct{}

func (p *YtDlpParser) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	YtdlCmd, err := global.GetConfig("ytdl_cmd")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	YtdlArgs, err := global.GetConfig("ytdl_args")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	ytdlArgs := strings.Fields(YtdlArgs)
	for i, v := range ytdlArgs {
		if strings.EqualFold(v, "{url}") {
			ytdlArgs[i] = channel.URL
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
		cleanLines := []string(nil)
		for _, l := range lines {
			if strings.HasPrefix(l, "http") {
				cleanLines = append(cleanLines, l)
			}
		}
		output = strings.Join(cleanLines, "\n")
		if err == nil || strings.HasSuffix(output, "m3u8") {
			li := &model.LiveInfo{}
			li.LiveUrl = output
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
	service.RegisterPlugin("yt-dlp", &YtDlpParser{}, 7)
}
