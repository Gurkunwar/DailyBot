package store

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
)

func InitRedis() (*redis.Client, error) {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "redis://localhost:6379/0"
	}

	opts, err := redis.ParseURL(addr)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opts)

	err = rdb.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}