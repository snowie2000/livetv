// youtube
package plugin

import (
	"context"
	"errors"
	"log"
	"os/exec"
	"strings"

	"github.com/snowie2000/livetv/model"

	"github.com/snowie2000/livetv/global"
)

type YtDlpParser struct{}

func (p *YtDlpParser) Parse(liveUrl string, proxyUrl string, lastInfo string) (*model.LiveInfo, error) {
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
		if err == nil {
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
	registerPlugin("yt-dlp", &YtDlpParser{})
}
