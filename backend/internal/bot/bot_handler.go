package bot

import (
	"log"
	"os"

	"github.com/Gurkunwar/dailybot/internal/bot/poll"
	"github.com/Gurkunwar/dailybot/internal/bot/standup"
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
	Standups       *standup.StandupHandler
	Polls          *poll.PollHandler
}

func NewBotHandler(session *discordgo.Session,
	redis *redis.Client,
	db *gorm.DB,
	standupService *services.StandupService,
	userService *services.UserService) *BotHanlder {

	standupHandler := standup.NewStandupHandler(db, redis, standupService)
	pollhandler := poll.NewPollHandler(db, redis)

	return &BotHanlder{
		Session:        session,
		Redis:          redis,
		DB:             db,
		StandupService: standupService,
		UserService:    userService,
		Standups:       standupHandler,
		Polls:          pollhandler,
	}
}

func (h *BotHanlder) OnInteraction(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	if h.Standups.StandupRouter(session, intr) {
		return
	}

	if h.Polls.PollRouter(session, intr) {
		return
	}

	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		switch intr.ApplicationCommandData().Name {
		case "help":
			h.handleHelp(session, intr)
		case "timezone":
			h.sendTimezoneMenu(session, intr, 0)
		case "delete-my-data":
			h.handleDeleteMyData(session, intr)
		}
	case discordgo.InteractionMessageComponent:
		if intr.MessageComponentData().CustomID == "select_tz" {
			h.handleTimezoneSelection(session, intr)
		}
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
