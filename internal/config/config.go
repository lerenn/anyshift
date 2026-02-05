package config

import (
	"os"
	"strconv"
)

// Config holds application configuration from environment.
type Config struct {
	GHToken         string
	DatabaseURL     string
	PollIntervalSec int
	HTTPAddr        string
	ConsumerWorkers int
	ChannelSize     int
}

// Default values when env vars are unset.
const (
	DefaultPollIntervalSec = 60
	DefaultHTTPAddr       = ":8080"
	DefaultConsumerWorkers = 3
	DefaultChannelSize    = 1000
)

// Load reads configuration from the environment.
// Uses defaults for optional values when unset.
func Load() *Config {
	c := &Config{
		GHToken:         os.Getenv("GH_TOKEN"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		PollIntervalSec: DefaultPollIntervalSec,
		HTTPAddr:        DefaultHTTPAddr,
		ConsumerWorkers: DefaultConsumerWorkers,
		ChannelSize:     DefaultChannelSize,
	}
	if v := os.Getenv("POLL_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.PollIntervalSec = n
		}
	}
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		c.HTTPAddr = v
	}
	if v := os.Getenv("CONSUMER_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.ConsumerWorkers = n
		}
	}
	if v := os.Getenv("CHANNEL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			c.ChannelSize = n
		}
	}
	return c
}
