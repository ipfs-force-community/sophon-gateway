package types

import (
	"time"
)

type RequestConfig struct {
	RequestQueueSize int
	RequestTimeout   time.Duration
	ClearInterval    time.Duration
}

func DefaultConfig() *RequestConfig {
	return &RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   time.Minute * 5,
		ClearInterval:    time.Minute * 5,
	}
}

type APIRegisterHubConfig struct {
	RegisterAPI     []string `json:"apiRegisterHub"`
	Token           string   `json:"token"`
	SupportAccounts []string `json:"supportAccounts"`
}
