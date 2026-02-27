package bot

import (
	"log"
	"os"

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

func NewSession() (*discordgo.Session, error) {
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))

	if err != nil {
		return nil, err
	}

	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentDirectMessages
    return dg, nil
}

func RegisterCommands(dg *discordgo.Session) {
	log.Println("Registering bot commands...")
    for _, command := range Commands {
        _, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", command)
        if err != nil {
            log.Printf("Cannot create '%v' command: %v", command.Name, err)
        }
    }
}
