package bot

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

func NewSession() (*discordgo.Session, error) {
	dg, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))

	if err != nil {
		return nil, err
	}

	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentDirectMessages
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