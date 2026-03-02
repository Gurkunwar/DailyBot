package poll

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) handleVote(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	customID := intr.MessageComponentData().CustomID
	userID := intr.Member.User.ID

	var pollID, optionID uint
	fmt.Sscanf(customID, "vote_%d_%d", &pollID, &optionID)

	var existingVote models.PollVote
	result := h.DB.Where("poll_id = ? AND user_id = ?", pollID, userID).First(&existingVote)

	if result.Error == nil {
		if existingVote.OptionID == optionID {
			h.DB.Unscoped().Delete(&existingVote)
		} else {
			existingVote.OptionID = optionID
			h.DB.Save(&existingVote)
		}
	} else {
		h.DB.Create(&models.PollVote{
			PollID:   pollID,
			OptionID: optionID,
			UserID:   userID,
		})
	}

	var poll models.Poll
	h.DB.Preload("Options").Where("id = ?", pollID).First(&poll)

	var totalVotes int64
	h.DB.Model(&models.PollVote{}).Where("poll_id = ?", pollID).Count(&totalVotes)

	var descriptionBuilder strings.Builder
	for _, opt := range poll.Options {
		var optVotes int64
		h.DB.Model(&models.PollVote{}).Where("option_id = ?", opt.ID).Count(&optVotes)

		percentage := 0.0
		if totalVotes > 0 {
			percentage = (float64(optVotes) / float64(totalVotes)) * 100
		}

		filled := int(percentage / 10)
		if filled == 0 && optVotes > 0 {
			filled = 1
		}
		empty := 10 - filled
		bar := strings.Repeat("🟦", filled) + strings.Repeat("⬛", empty)

		descriptionBuilder.WriteString(fmt.Sprintf("**%s**\n> %s %d votes (%.0f%%)\n\n",
			opt.Label, bar, optVotes, percentage))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "📊 " + poll.Question,
		Description: descriptionBuilder.String(),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total Votes: %d", totalVotes),
		},
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: intr.Message.Components,
		},
	})
}
