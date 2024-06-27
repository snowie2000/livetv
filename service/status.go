package service

import (
	"time"

	"github.com/zjyl1994/livetv/syncx"
)

type StatusInfo struct {
	Time               time.Time
	CoolDownMultiplier int
	RetryCount         int
	Status             int
	Msg                string
}

const (
	Unknown = iota
	Ok
	Warning
	Error
	Expired
)

const MaxRetryCount = 5

var statusCache syncx.Map[any, *StatusInfo]

func UpdateStatus(url any, status int, msg string) {
	if c, ok := statusCache.Load(url); ok {
		c.Msg = msg
		c.Status = status
		c.Time = time.Now()
		if status == Ok {
			c.RetryCount = 0
			c.CoolDownMultiplier = 1
		} else {
			c.RetryCount++
		}
	} else {
		statusCache.Store(url, &StatusInfo{
			Msg:                msg,
			Status:             status,
			RetryCount:         0,
			CoolDownMultiplier: 1,
			Time:               time.Now(),
		})
	}
}

func GetStatus(url any) *StatusInfo {
	if c, ok := statusCache.Load(url); ok {
		return c
	} else {
		return &StatusInfo{
			Status:             Unknown,
			Msg:                "Not yet parsed",
			Time:               time.Time{},
			CoolDownMultiplier: 1,
			RetryCount:         0,
		}
	}
}

func DeleteStatus(url any) {
	statusCache.Delete(url)
}
