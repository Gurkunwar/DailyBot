package poll

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type PollHandler struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func NewPollHandler(db *gorm.DB, redis *redis.Client) *PollHandler {
	return &PollHandler{DB: db, Redis: redis}
}
