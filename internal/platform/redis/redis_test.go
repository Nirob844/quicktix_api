package redis

import "testing"

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		Addr:     "localhost:6380",
		Password: "",
		DB:       0,
	}

	if cfg.Addr != "localhost:6380" {
		t.Errorf("expected Addr to be localhost:6380, got %s", cfg.Addr)
	}
}

