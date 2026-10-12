package service

import (
	"sync"
	"time"

	"github.com/snowie2000/livetv/syncx"
)

type StatusInfo struct {
	mu                 sync.RWMutex
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
	c, ok := statusCache.Load(url)
	if !ok {
		newInfo := &StatusInfo{
			Msg:                msg,
			Status:             status,
			RetryCount:         0,
			CoolDownMultiplier: 1,
			Time:               time.Now(),
		}
		actual, loaded := statusCache.LoadOrStore(url, newInfo)
		if !loaded {
			return
		}
		c = actual
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.Msg = msg
	c.Status = status
	c.Time = time.Now()
	if status == Ok {
		c.RetryCount = 0
		c.CoolDownMultiplier = 1
	} else {
		c.RetryCount++
	}
}

func GetStatus(url any) *StatusInfo {
	if c, ok := statusCache.Load(url); ok {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return &StatusInfo{
			Time:               c.Time,
			CoolDownMultiplier: c.CoolDownMultiplier,
			RetryCount:         c.RetryCount,
			Status:             c.Status,
			Msg:                c.Msg,
		}
	}
	return &StatusInfo{
		Status:             Unknown,
		Msg:                "Not yet parsed",
		Time:               time.Time{},
		CoolDownMultiplier: 1,
		RetryCount:         0,
	}
}

func DeleteStatus(url any) {
	statusCache.Delete(url)
}
