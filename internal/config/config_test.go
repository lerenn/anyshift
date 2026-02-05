package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()
	if cfg.PollIntervalSec != DefaultPollIntervalSec {
		t.Errorf("PollIntervalSec want %d got %d", DefaultPollIntervalSec, cfg.PollIntervalSec)
	}
	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Errorf("HTTPAddr want %s got %s", DefaultHTTPAddr, cfg.HTTPAddr)
	}
	if cfg.ConsumerWorkers != DefaultConsumerWorkers {
		t.Errorf("ConsumerWorkers want %d got %d", DefaultConsumerWorkers, cfg.ConsumerWorkers)
	}
	if cfg.ChannelSize != DefaultChannelSize {
		t.Errorf("ChannelSize want %d got %d", DefaultChannelSize, cfg.ChannelSize)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	os.Clearenv()
	os.Setenv("POLL_INTERVAL_SEC", "120")
	os.Setenv("HTTP_ADDR", ":9090")
	os.Setenv("CONSUMER_WORKERS", "5")
	os.Setenv("CHANNEL_SIZE", "500")
	os.Setenv("GH_TOKEN", "secret")
	os.Setenv("DATABASE_URL", "postgres://local/db")
	cfg := Load()
	if cfg.PollIntervalSec != 120 {
		t.Errorf("PollIntervalSec want 120 got %d", cfg.PollIntervalSec)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr want :9090 got %s", cfg.HTTPAddr)
	}
	if cfg.ConsumerWorkers != 5 {
		t.Errorf("ConsumerWorkers want 5 got %d", cfg.ConsumerWorkers)
	}
	if cfg.ChannelSize != 500 {
		t.Errorf("ChannelSize want 500 got %d", cfg.ChannelSize)
	}
	if cfg.GHToken != "secret" {
		t.Errorf("GHToken want secret got %s", cfg.GHToken)
	}
	if cfg.DatabaseURL != "postgres://local/db" {
		t.Errorf("DatabaseURL want postgres://local/db got %s", cfg.DatabaseURL)
	}
}

func TestLoad_InvalidValuesUseDefaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("POLL_INTERVAL_SEC", "invalid")
	os.Setenv("CONSUMER_WORKERS", "0")
	os.Setenv("CHANNEL_SIZE", "-1")
	cfg := Load()
	if cfg.PollIntervalSec != DefaultPollIntervalSec {
		t.Errorf("PollIntervalSec want default %d got %d", DefaultPollIntervalSec, cfg.PollIntervalSec)
	}
	if cfg.ConsumerWorkers != DefaultConsumerWorkers {
		t.Errorf("ConsumerWorkers want default %d got %d", DefaultConsumerWorkers, cfg.ConsumerWorkers)
	}
	if cfg.ChannelSize != DefaultChannelSize {
		t.Errorf("ChannelSize want default %d got %d", DefaultChannelSize, cfg.ChannelSize)
	}
}
