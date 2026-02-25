package bot

import (
	"github.com/Gurkunwar/dailybot/internal/services"
	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BotHanlder struct {
	Session        *discordgo.Session
	Redis          *redis.Client
	DB             *gorm.DB
	StandupService *services.StandupService
	UserService    *services.UserService
}

func NewBotHandler(session *discordgo.Session,
	redis *redis.Client,
	db *gorm.DB,
	standupService *services.StandupService, 
	userService *services.UserService) *BotHanlder {

	return &BotHanlder{
		Session:        session,
		Redis:          redis,
		DB:             db,
		StandupService: standupService,
		UserService: userService,
	}
}
