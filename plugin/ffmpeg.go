package plugin

import (
	"encoding/json"
	"github.com/snowie2000/livetv/service"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nareix/joy5/av"
	"github.com/snowie2000/livetv/global"
	"github.com/snowie2000/livetv/model"

	"github.com/nareix/joy5/format/flv"
	"github.com/nareix/joy5/format/rtmp"
)

type TranscodeParser struct {
	URLM3U8Parser
}

func (p *TranscodeParser) Host(c *gin.Context, info *model.LiveInfo, chInfo *model.Channel) error {
	rtmpConn, conn, err := rtmp.NewClient().Dial(info.LiveUrl, rtmp.PrepareReading)
	if err != nil {
		return err
	}
	log.Println("Start transcoding", info.LiveUrl)
	defer conn.Close()
	defer log.Println("Transcoding finished")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Header().Set("Access-Control-Allow-Methods", "*")
	c.Writer.Header().Set("Content-Type", "video/x-flv")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.WriteHeader(200)
	c.Writer.Flush()

	muxer := flv.NewMuxer(c.Writer)
	err = muxer.WriteFileHeader()
	var packet av.Packet
	for err == nil {
		packet, err = rtmpConn.ReadPacket()
		if err != nil {
			log.Println("stream ended with error", err)
			break
		}
		err = muxer.WritePacket(packet)
	}
	return nil
}

func (p *TranscodeParser) Parse(channel *model.Channel, prevLiveInfo *model.LiveInfo) (*model.LiveInfo, error) {
	previousExtraInfo := prevLiveInfo.ExtraInfo
	u, err := url.Parse(channel.URL)
	if err != nil || !strings.EqualFold(u.Scheme, "rtmp") {
		client := http.Client{
			Timeout:   time.Second * 10,
			Transport: global.TransportWithProxy(channel.ProxyUrl),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Jar: global.CookieJar,
		}
		req, err := http.NewRequest("GET", channel.URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", service.DefaultUserAgent)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer global.CloseBody(resp)
		var ui service.UrlInfo
		decoder := json.NewDecoder(resp.Body)
		if decoder.Decode(&ui) == nil && len(ui.Headers) > 0 {
			js, _ := json.Marshal(ui)
			previousExtraInfo = string(js) // write headers info to extraInfo
		}

		redir := resp.Header.Get("Location")
		if redir == "" {
			return nil, service.NoMatchFeed
		}
		u, err = url.Parse(redir)
	}
	if err == nil && strings.EqualFold(u.Scheme, "rtmp") {
		li := &model.LiveInfo{}
		li.LiveUrl = u.String()
		li.ExtraInfo = previousExtraInfo
		return li, nil
	}
	return nil, service.NoMatchFeed
}

func init() {
	service.RegisterPlugin("transcode", &TranscodeParser{}, 3)
}
