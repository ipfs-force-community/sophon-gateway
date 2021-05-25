package types

import "time"

type Config struct {
	RequestQueueSize int
	RequestTimeout   time.Duration
}
