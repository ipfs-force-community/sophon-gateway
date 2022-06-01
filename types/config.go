package types

import (
	"time"
)

type Config struct {
	RequestQueueSize int
	RequestTimeout   time.Duration
	ClearInterval    time.Duration
}

func DefaultConfig() *Config {
	return &Config{
		RequestQueueSize: 30,
		RequestTimeout:   time.Minute * 5,
		ClearInterval:    time.Minute * 5,
	}
}
