package model

type Channel struct {
	ID            int    `gorm:"primary_key"`
	ChannelID     string `gorm:"-:all"`
	Name          string
	Logo          string
	URL           string
	Parser        string
	Proxy         bool
	TsProxy       string     // new field for customized live.ts server
	ProxyUrl      string     // proxy for server connection
	Token         string     `gorm:"-:all"`
	Category      string     `gorm:"index"`
	HasSubChannel bool       `gorm:"hassubchn"`
	Children      []*Channel `gorm: "-:all` // sub channel list
}

type LiveInfo struct {
	LiveUrl   string
	Logo      string
	ExtraInfo string
}
