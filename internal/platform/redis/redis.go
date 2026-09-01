package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds configuration parameters for Redis connection.
type Config struct {
	Addr     string
	Password string
	DB       int
}

// Connect initializes and returns a Redis client after testing connection with Ping.
func Connect(cfg Config) (*redis.Client, error) {
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6380"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis at %s: %w", cfg.Addr, err)
	}

	return rdb, nil
}

