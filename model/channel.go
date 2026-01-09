package model

import (
	"crypto/md5"
	"encoding/hex"
)

type Channel struct {
	ID                int    `gorm:"primary_key"`
	ParentID          string `gorm:"-:all"`
	ChannelID         string `gorm:"-:all"`
	Name              string
	Logo              string
	URL               string
	Parser            string
	Proxy             bool
	TsProxy           string     // new field for customized live.ts server
	ProxyUrl          string     // proxy for server connection
	Token             string     `gorm:"-:all"`
	CustomQueryString string     `gorm:"-:all"` // custom extra url query param
	Category          string     `gorm:"index"`
	HasSubChannel     bool       `gorm:"hassubchn"`
	Extra             string     // same as information returned from redirection page
	Children          []*Channel `gorm:"-:all"` // sub channel list
}

func (c *Channel) Digest() string {
	// return md5 of c.Name
	hash := md5.Sum([]byte(c.ParentID + c.Name))
	return hex.EncodeToString(hash[:])
}

type LiveInfo struct {
	LiveUrl   string
	Logo      string
	ExtraInfo string
}
