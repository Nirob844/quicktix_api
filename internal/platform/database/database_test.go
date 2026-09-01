package database

import (
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		DSN:             "postgres://user:pass@localhost:5432/db",
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}

	if cfg.MaxOpenConns != 10 {
		t.Errorf("expected MaxOpenConns to be 10, got %d", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 2 {
		t.Errorf("expected MaxIdleConns to be 2, got %d", cfg.MaxIdleConns)
	}
}

