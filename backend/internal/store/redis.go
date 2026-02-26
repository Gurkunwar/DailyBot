package store

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/redis/go-redis/v9"
)

func InitRedis() (*redis.Client, error) {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
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

func SaveState(rdb *redis.Client, userID string, state models.StandupState) {
	data, _ := json.Marshal(state)
	rdb.Set(context.Background(), "state:" + userID, data, 24*time.Hour)
}

func GetState(rdb *redis.Client, userID string) (*models.StandupState, error) {
	val, err := rdb.Get(context.Background(), "state:" + userID).Result()
	if err != nil {
		return nil, err
	}

	var state models.StandupState
	json.Unmarshal([]byte(val), &state)

	return &state, nil
}