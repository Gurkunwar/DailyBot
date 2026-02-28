package standup

import (
    "github.com/Gurkunwar/dailybot/internal/services"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"
)

type StandupHandler struct {
    DB             *gorm.DB
    Redis          *redis.Client
    StandupService *services.StandupService
}

func NewStandupHandler(db *gorm.DB, redis *redis.Client, svc *services.StandupService) *StandupHandler {
    return &StandupHandler{
        DB:             db,
        Redis:          redis,
        StandupService: svc,
    }
}