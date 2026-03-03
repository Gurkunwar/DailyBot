package poll

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/dailybot/internal/bot/utils"
	"github.com/Gurkunwar/dailybot/internal/models"
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

	var pollAnswers []discordgo.PollAnswer

	for i := 1; i <= 5; i++ {
		optName := fmt.Sprintf("option_%d", i)
		if opt, ok := optionMap[optName]; ok && opt.StringValue() != "" {
			pollAnswers = append(pollAnswers, discordgo.PollAnswer{
				Media: &discordgo.PollMedia{ // Pointer here!
					Text: strings.TrimSpace(opt.StringValue()),
				},
			})
		}
	}

	nativePoll := &discordgo.Poll{
		Question: discordgo.PollMedia{
			Text: questionText,
		},
		Answers:          pollAnswers,
		AllowMultiselect: false,
		Duration:         durationHours,
	}

	msg, err := session.ChannelMessageSendComplex(intr.ChannelID, &discordgo.MessageSend{
		Poll: nativePoll,
	})

	if err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Failed to publish native poll.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	pollModel := models.Poll{
		GuildID:   intr.GuildID,
		ChannelID: intr.ChannelID,
		CreatorID: userID,
		Question:  questionText,
		MessageID: msg.ID,
		IsActive:  true,
	}
	h.DB.Create(&pollModel)

	for _, answer := range pollAnswers {
		h.DB.Create(&models.PollOption{
			PollID: pollModel.ID,
			Label:  answer.Media.Text,
		})
	}

	receiptMessage := fmt.Sprintf("✅ Poll published! (Poll ID: `%d`)", pollModel.ID)
    session.ChannelMessageSend(intr.ChannelID, receiptMessage)

    session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "Poll created successfully.",
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

	// 1. Get the Poll from your local DB to find the ChannelID and MessageID
	var poll models.Poll
	if err := h.DB.First(&poll, pollID).Error; err != nil {
		utils.RespondWithMessage(session, intr, "❌ Poll not found in your database. Please check the ID.", true)
		return
	}

	// 2. Fetch the actual message from Discord
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

	// 3. Loop through the Native Poll answers
	for _, answer := range msg.Poll.Answers {
		report.WriteString(fmt.Sprintf("**%s**\n", answer.Media.Text))

		// Fetch voters directly from Discord API for this specific option
		// We fetch up to 100 voters. The "" is for pagination (after a specific user ID).
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

	// 4. Send the report back to the admin privately
	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: report.String(),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
