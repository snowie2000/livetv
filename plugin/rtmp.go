package plugin

import (
	"encoding/json"
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

type RTMPParser struct {
	URLM3U8Parser
}

func (p *RTMPParser) Host(c *gin.Context, info *model.LiveInfo) error {
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

func (p *RTMPParser) Parse(liveUrl string, proxyUrl string, previousExtraInfo string) (*model.LiveInfo, error) {
	u, err := url.Parse(liveUrl)
	if err != nil || !strings.EqualFold(u.Scheme, "rtmp") {
		client := http.Client{
			Timeout:   time.Second * 10,
			Transport: global.TransportWithProxy(proxyUrl),
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
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
		var ui UrlInfo
		decoder := json.NewDecoder(resp.Body)
		if decoder.Decode(&ui) == nil && len(ui.Headers) > 0 {
			js, _ := json.Marshal(ui)
			previousExtraInfo = string(js) // write headers info to extraInfo
		}

		redir := resp.Header.Get("Location")
		if redir == "" {
			return nil, NoMatchFeed
		}
		u, err = url.Parse(redir)
	}
	if err == nil && strings.EqualFold(u.Scheme, "rtmp") {
		li := &model.LiveInfo{}
		li.LiveUrl = u.String()
		li.ExtraInfo = previousExtraInfo
		return li, nil
	}
	return nil, NoMatchFeed
}

func init() {
	registerPlugin("rtmp", &RTMPParser{}, 3)
}
