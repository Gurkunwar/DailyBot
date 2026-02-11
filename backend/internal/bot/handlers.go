package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BotHanlder struct {
	Session *discordgo.Session
	Redis   *redis.Client
	DB      *gorm.DB
}