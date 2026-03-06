package poll

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/asyncflow/internal/bot/utils"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) handleCreateNativePoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
    userID := intr.Member.User.ID
    options := intr.ApplicationCommandData().Options

    optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
    for _, opt := range options {
        optionMap[opt.Name] = opt
    }

    questionText := optionMap["question"].StringValue()

    durationHours := 24
    if opt, ok := optionMap["duration"]; ok {
        durationHours = int(opt.IntValue())
    }

    var strOptions []string
    for i := 1; i <= 5; i++ {
        optName := fmt.Sprintf("option_%d", i)
        if opt, ok := optionMap[optName]; ok && opt.StringValue() != "" {
            strOptions = append(strOptions, opt.StringValue())
        }
    }

    pollModel, err := h.Service.CreatePoll(
        intr.GuildID,
        intr.ChannelID,
        userID,
        questionText,
        strOptions,
        durationHours,
    )

    if err != nil {
        session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: fmt.Sprintf("❌ Failed to publish native poll: %v", err),
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }


    session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: fmt.Sprintf("✅ Poll created successfully! (Poll ID: `%d`)", pollModel.ID),
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
}

func (h *PollHandler) HandlePollAudit(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	if !utils.IsServerAdmin(intr) {
		utils.RespondWithMessage(session, intr, "⛔ This command is reserved for Server Admins.", true)
		return
	}

	options := intr.ApplicationCommandData().Options
	pollID := options[0].IntValue()

	var poll models.Poll
	if err := h.DB.First(&poll, pollID).Error; err != nil {
		utils.RespondWithMessage(session, intr, "❌ Poll not found in your database. Please check the ID.", true)
		return
	}

	msg, err := session.ChannelMessage(poll.ChannelID, poll.MessageID)
	if err != nil || msg.Poll == nil {
		utils.RespondWithMessage(session, intr,
			"❌ Could not find the active native poll on Discord. (It may have been deleted).",
			true)
		return
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("📋 **Audit Report: %s**\n", msg.Poll.Question.Text))
	report.WriteString(fmt.Sprintf("_Poll ID: %d | Live Data from Discord API_\n\n", poll.ID))

	for _, answer := range msg.Poll.Answers {
		report.WriteString(fmt.Sprintf("**%s**\n", answer.Media.Text))
		voters, err := session.PollAnswerVoters(poll.ChannelID, poll.MessageID, answer.AnswerID)

		if err != nil || len(voters) == 0 {
			report.WriteString("> _No votes cast for this option._\n\n")
		} else {
			for _, voter := range voters {
				report.WriteString(fmt.Sprintf("> • <@%s>\n", voter.ID))
			}
			report.WriteString("\n")
		}
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: report.String(),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
