package poll

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/dailybot/internal/bot/utils"
	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) handlePublishPoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := intr.Member.User.ID
	state, err := store.GetPollDraft(h.Redis, userID)

	if err != nil || state.Question == "" || len(state.Options) < 2 {
		utils.RespondWithMessage(session, intr, "❌ Your poll is incomplete!", true)
		return
	}

	poll := models.Poll{
		GuildID:   intr.GuildID,
		ChannelID: intr.ChannelID,
		CreatorID: userID,
		Question:  state.Question,
	}

	h.DB.Create(&poll)

	var buttons []discordgo.MessageComponent
	var descriptionBuilder strings.Builder

	for _, optText := range state.Options {
		option := models.PollOption{
			PollID: poll.ID,
			Label:  optText,
		}
		h.DB.Create(&option)

		emptyBar := strings.Repeat("⬛", 10)
		descriptionBuilder.WriteString(fmt.Sprintf("**%s**\n> %s 0 votes (0%%)\n\n", option.Label, emptyBar))

		buttons = append(buttons, discordgo.Button{
			Label:    option.Label,
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("vote_%d_%d", poll.ID, option.ID),
		})
	}

	var rows []discordgo.MessageComponent
	for i := 0; i < len(buttons); i += 5 {
		end := i + 5
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, discordgo.ActionsRow{Components: buttons[i:end]})
	}

	editQuestionBtn := discordgo.Button{
		Label:    "✏️ Edit Question",
		Style:    discordgo.SecondaryButton,
		CustomID: fmt.Sprintf("poll_btn_edit_%d", poll.ID),
	}

	endPollBtn := discordgo.Button{
		Label:    "🛑 End Poll",
		Style:    discordgo.DangerButton,
		CustomID: fmt.Sprintf("poll_btn_end_%d", poll.ID),
	}

	rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{editQuestionBtn, endPollBtn}})

	embed := &discordgo.MessageEmbed{
		Title:       "📊 " + poll.Question,
		Description: descriptionBuilder.String(),
		Color:       0x5865F2,
		Footer: &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("ID: %d • Created by %s • Total Votes: 0",
			poll.ID, intr.Member.User.Username)},
	}

	msg, err := session.ChannelMessageSendComplex(intr.ChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed}, Components: rows,
	})

	if err != nil {
		utils.RespondWithMessage(session, intr, "❌ Failed to publish poll.", true)
		return
	}

	poll.MessageID = msg.ID
	h.DB.Save(&poll)
	store.ClearPollDraft(h.Redis, userID)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "✅ **Poll successfully published!**",
			Embeds:     nil,
			Components: nil,
		},
	})
}

func (h *PollHandler) handleCancelPoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := intr.Member.User.ID
	store.ClearPollDraft(h.Redis, userID)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "🚫 **Poll creation cancelled.**",
			Embeds:     nil,
			Components: nil,
		},
	})
}

func (h *PollHandler) handleEditPoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	customID := intr.MessageComponentData().CustomID
	userID := intr.Member.User.ID

	var pollID uint
	fmt.Sscanf(customID, "poll_btn_edit_%d", &pollID)

	var poll models.Poll
	if err := h.DB.Where("id = ?", pollID).First(&poll).Error; err != nil {
		utils.RespondWithMessage(session, intr, "❌ Poll not found.", true)
		return
	}

	if poll.CreatorID != userID && !utils.IsServerAdmin(intr) {
		utils.RespondWithMessage(session, intr, "⛔ Only the creator can edit this poll.", true)
		return
	}
	if !poll.IsActive {
		utils.RespondWithMessage(session, intr, "⚠️ You cannot edit a closed poll.", true)
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("poll_modal_edit_%d", poll.ID),
			Title:    "Edit Poll Question",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "new_question_text",
							Label:     "Update your question:",
							Style:     discordgo.TextInputShort,
							Required:  true,
							MaxLength: 250,
							Value:     poll.Question,
						},
					},
				},
			},
		},
	})
}

func (h *PollHandler) saveEditedQuestion(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	customID := intr.ModalSubmitData().CustomID

	var pollID uint
	fmt.Sscanf(customID, "poll_modal_edit_%d", &pollID)

	newQuestionText := intr.ModalSubmitData().
		Components[0].(*discordgo.ActionsRow).
		Components[0].(*discordgo.TextInput).Value
	newQuestionText = strings.TrimSpace(newQuestionText)

	var poll models.Poll
	h.DB.Preload("Options").Where("id = ?", pollID).First(&poll)
	poll.Question = newQuestionText
	h.DB.Save(&poll)

	var totalVotes int64
	h.DB.Model(&models.PollVote{}).Where("poll_id = ?", poll.ID).Count(&totalVotes)

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
			Text: fmt.Sprintf("ID: %d • Total Votes: %d (Edited)", poll.ID, totalVotes),
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

func (h *PollHandler) handleEndPoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	customID := intr.MessageComponentData().CustomID
	userID := intr.Member.User.ID

	var pollID uint
	fmt.Sscanf(customID, "poll_btn_end_%d", &pollID)

	var poll models.Poll
	if err := h.DB.Preload("Options").Where("id = ?", pollID).First(&poll).Error; err != nil {
		utils.RespondWithMessage(session, intr, "❌ Poll not found in database.", true)
		return
	}

	if poll.CreatorID != userID && !utils.IsServerAdmin(intr) {
		utils.RespondWithMessage(session, intr, "⛔ Only the creator of this poll can end it.", true)
		return
	}

	if !poll.IsActive {
		utils.RespondWithMessage(session, intr, "⚠️ This poll has already been closed.", true)
		return
	}

	poll.IsActive = false
	h.DB.Save(&poll)

	var totalVotes int64
	h.DB.Model(&models.PollVote{}).Where("poll_id = ?", poll.ID).Count(&totalVotes)

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
		Title:       "🔒 [CLOSED] " + poll.Question,
		Description: descriptionBuilder.String(),
		Color:       0xED4245,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("ID: %d • Closed by %s • Final Total Votes: %d",
                poll.ID, intr.Member.User.Username, totalVotes),
		},
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (h *PollHandler) HandlePollAudit(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	// userID := intr.Member.User.ID

	if !utils.IsServerAdmin(intr) {
		utils.RespondWithMessage(session, intr, "⛔ This command is reserved for Server Admins.", true)
		return
	}

	options := intr.ApplicationCommandData().Options
	pollID := options[0].IntValue()

	var poll models.Poll
	if err := h.DB.Preload("Options").First(&poll, pollID).Error; err != nil {
		utils.RespondWithMessage(session, intr, "❌ Poll not found. Please check the ID.", true)
		return
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("📋 **Audit Report: %s**\n", poll.Question))
	report.WriteString(fmt.Sprintf("_Poll ID: %d | Total Votes logged in DB_\n\n", poll.ID))

	for _, opt := range poll.Options {
		var votes []models.PollVote
		h.DB.Where("poll_id = ? AND option_id = ?", poll.ID, opt.ID).Find(&votes)

		report.WriteString(fmt.Sprintf("**%s** (%d votes)\n", opt.Label, len(votes)))

		if len(votes) == 0 {
			report.WriteString("> _No votes cast for this option._\n")
		} else {
			for _, v := range votes {
				report.WriteString(fmt.Sprintf("> • <@%s>\n", v.UserID))
			}
		}
		report.WriteString("\n")
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: report.String(),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
