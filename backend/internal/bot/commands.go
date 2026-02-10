package bot

import (
	"fmt"
	"regexp"
	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) handleSetChannel(s *discordgo.Session, m *discordgo.MessageCreate) {
	re := regexp.MustCompile(`<#(\d+)>`)
	match := re.FindStringSubmatch(m.Content)

	if len(match) > 1 {
		targetChannelID := match[1]
		var guild models.Guild

		h.DB.Where(&models.Guild{GuildID: m.GuildID}).FirstOrCreate(&guild)
		guild.ReportChannelID = targetChannelID
		h.DB.Save(&guild)

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Success! Daily reports will now be sent to <#%s>", targetChannelID))
		return
	}

	s.ChannelMessageSend(m.ChannelID, "Please mention a channel, e.g., `\\set-channel #general`.")
}