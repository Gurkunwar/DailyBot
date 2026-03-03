package poll

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/dailybot/internal/bot/utils"
	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) HandlePollEnd(session *discordgo.Session, intr *discordgo.InteractionCreate) {
    if !utils.IsServerAdmin(intr) {
        utils.RespondWithMessage(session, intr, "⛔ This command is reserved for Server Admins.", true)
        return
    }

    pollID := intr.ApplicationCommandData().Options[0].IntValue()
    var poll models.Poll
    if err := h.DB.First(&poll, pollID).Error; err != nil {
        utils.RespondWithMessage(session, intr, "❌ Poll not found in DB.", true)
        return
    }

    endpoint := discordgo.EndpointChannel(poll.ChannelID) + "/polls/" + poll.MessageID + "/expire"
    body := map[string]interface{}{}

    _, err := session.RequestWithBucketID("POST", endpoint, body, discordgo.EndpointChannelMessage(poll.ChannelID, ""))
    
    if err != nil {
        utils.RespondWithMessage(session, intr, fmt.Sprintf("❌ Failed to end poll. Error: %v", err), true)
        return
    }

    utils.RespondWithMessage(session, intr, "✅ **Poll has been successfully closed!**", true)
}

func (h *PollHandler) HandlePollExport(session *discordgo.Session, intr *discordgo.InteractionCreate) {
    if !utils.IsServerAdmin(intr) {
        utils.RespondWithMessage(session, intr, "⛔ Admin only.", true)
        return
    }

    pollID := intr.ApplicationCommandData().Options[0].IntValue()
    var poll models.Poll
    if err := h.DB.First(&poll, pollID).Error; err != nil {
        utils.RespondWithMessage(session, intr, "❌ Poll not found.", true)
        return
    }

    msg, err := session.ChannelMessage(poll.ChannelID, poll.MessageID)
    if err != nil || msg.Poll == nil {
        utils.RespondWithMessage(session, intr, "❌ Could not fetch poll from Discord.", true)
        return
    }

    var csvBuilder strings.Builder
    csvBuilder.WriteString("Option,User ID,Username\n")

    for _, answer := range msg.Poll.Answers {
        optionText := strings.ReplaceAll(answer.Media.Text, ",", ";")
        
        voters, _ := session.PollAnswerVoters(poll.ChannelID, poll.MessageID, answer.AnswerID)
        
        if len(voters) == 0 {
            csvBuilder.WriteString(fmt.Sprintf("%s,NONE,No votes\n", optionText))
        } else {
            for _, voter := range voters {
                csvBuilder.WriteString(fmt.Sprintf("%s,%s,%s\n", optionText, voter.ID, voter.Username))
            }
        }
    }

    file := &discordgo.File{
        Name:        fmt.Sprintf("poll_results_%d.csv", poll.ID),
        ContentType: "text/csv",
        Reader:      strings.NewReader(csvBuilder.String()),
    }

    session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "📊 **Here is your Excel/CSV Export!**",
            Files:   []*discordgo.File{file},
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
}