package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/redis/go-redis/v9"
)

func SavePollDraft(rdb *redis.Client, userID string, state models.PollState) {
	data, _ := json.Marshal(state)
	rdb.Set(context.Background(), "poll_draft:"+userID, data, time.Hour)
}

func GetPollDraft(rdb *redis.Client, userID string) (*models.PollState, error) {
	val, err := rdb.Get(context.Background(), "poll_draft:" + userID).Result()
	if err != nil {
		return nil, err
	}

	var state models.PollState
	json.Unmarshal([]byte(val), &state)

	return &state, nil
}

func ClearPollDraft(rdb *redis.Client, userID string) {
	rdb.Del(context.Background(), "poll_draft:"+userID)
}