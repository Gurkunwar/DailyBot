package poll

import (
	"fmt"

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

    description, totalVotes := h.renderPollDescription(poll)

    session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseUpdateMessage,
        Data: &discordgo.InteractionResponseData{
            Embeds: []*discordgo.MessageEmbed{{
                Title:       "📊 " + poll.Question,
                Description: description,
                Color:       0x5865F2,
                Footer: &discordgo.MessageEmbedFooter{
                    Text: fmt.Sprintf("ID: %d • Total Votes: %d", poll.ID, totalVotes),
                },
            }},
            Components: intr.Message.Components,
        },
    })
}