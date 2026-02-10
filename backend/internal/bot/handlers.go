package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BotHanlder struct {
	Session *discordgo.Session
	Redis   *redis.Client
	DB      *gorm.DB
}

func (h *BotHanlder) OnMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	switch {
	case strings.HasPrefix(m.Content, "\\set-channel"):
		h.handleSetChannel(s, m)
	case m.Content == "\\start":
		h.initiateStandup(s, m.Author.ID, m.GuildID)
	}
}
