package standup

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Gurkunwar/dailybot/internal/bot/utils"
	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *StandupHandler) InitiateStandup(s *discordgo.Session, userID string, guildID, channelID string, standupID uint) {
	var profile models.UserProfile
	h.DB.Unscoped().Preload("Standups").Where("user_id = ?", userID).First(&profile)

	if profile.ID != 0 && profile.DeletedAt.Valid {
		h.DB.Model(&profile).Unscoped().Update("deleted_at", nil)
	}

	targetChannelID := channelID
	if targetChannelID == "" {
		dm, _ := s.UserChannelCreate(userID)
		targetChannelID = dm.ID
	}

	if profile.ID == 0 || len(profile.Standups) == 0 {

		s.ChannelMessageSend(targetChannelID,
			"⛔ You are not part of any standups yet. Please ask your manager to add you.")
		return
	}

	var targetStandup models.Standup

	if standupID != 0 {
		isMember := false
		for _, st := range profile.Standups {
			if st.ID == standupID {
				targetStandup = st
				isMember = true
				break
			}
		}
		if !isMember {
			return
		}
	} else {
		if len(profile.Standups) == 1 {
			targetStandup = profile.Standups[0]
		} else {
			h.sendStandupSelectionMenu(s, userID, guildID, channelID, profile.Standups)
			return
		}
	}

	if targetStandup.ReportChannelID == "" {
		s.ChannelMessageSend(targetChannelID, "⚠️ This standup has no report channel set.")
		return
	}

	if profile.Timezone == "" {
		newTimezone := "UTC"

		if targetStandup.ManagerID != "" {
			var manager models.UserProfile
			if err := h.DB.Where("user_id = ?", targetStandup.ManagerID).First(&manager).Error; err == nil {
				if manager.Timezone != "" {
					newTimezone = manager.Timezone
				}
			}
		}

		profile.Timezone = newTimezone
		h.DB.Save(&profile)
	}

	channel, _ := s.UserChannelCreate(userID)

	if profile.Timezone == "UTC" {
		s.ChannelMessageSend(targetChannelID,
			"ℹ️ *Note: Daily reminders are scheduled in UTC. Use `/timezone` to change.*")
	}

	h.startQuestionFlow(s, channel.ID, userID, targetStandup)
}

func (h *StandupHandler) startQuestionFlow(session *discordgo.Session, channelID, userID string, standup models.Standup) {
	state := models.StandupState{
		UserID:    userID,
		GuildID:   standup.GuildID,
		StandupID: standup.ID,
		Answers:   []string{},
	}

	redisKey := fmt.Sprintf("%s_%d", userID, standup.ID)
	store.SaveState(h.Redis, redisKey, state)

	session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "Ready to submit your daily standup?",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Fill Standup",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("open_standup_modal_%d", standup.ID),
					},
					discordgo.Button{
						Label:    "⏭️ Skip Today",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("skip_standup_%d", standup.ID),
					},
				},
			},
		},
	})
}

func (h *StandupHandler) openSingleQuestionModal(session *discordgo.Session,
	intr *discordgo.InteractionCreate, qNum int) {

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("create_q_modal_%d", qNum),
			Title:    fmt.Sprintf("Standup Setup (Question %d)", qNum),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "question_text",
							Label:       fmt.Sprintf("Type Question %d", qNum),
							Style:       discordgo.TextInputShort,
							Required:    true,
							Placeholder: "e.g., What did you accomplish yesterday?",
						},
					},
				},
			},
		},
	})

	if err != nil {
		log.Println("Error opening single question modal:", err)
	}
}

func (h *StandupHandler) handleCreateQuestionSubmit(session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	customID string) {

	var currentQNum int
	fmt.Sscanf(customID, "create_q_modal_%d", &currentQNum)

	newQuestion := intr.ModalSubmitData().
		Components[0].(*discordgo.ActionsRow).
		Components[0].(*discordgo.TextInput).Value

	state, err := store.GetState(h.Redis, intr.Member.User.ID+"_create")
	if err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Session expired. Please start over.",
				Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}
	state.Answers = append(state.Answers, strings.TrimSpace(newQuestion))
	store.SaveState(h.Redis, intr.Member.User.ID+"_create", *state)

	responseType := discordgo.InteractionResponseChannelMessageWithSource
	if currentQNum > 1 {
		responseType = discordgo.InteractionResponseUpdateMessage
	}

	nextQNum := currentQNum + 1

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: responseType,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ **Question %d saved!**\n> %s\n\nDo you want to add Question %d, or finish creating the team?", currentQNum, newQuestion, nextQNum),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    fmt.Sprintf("➕ Add Question %d", nextQNum),
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("add_next_q_%d", nextQNum),
						},
						discordgo.Button{
							Label:    "🚀 Finish & Create",
							Style:    discordgo.SuccessButton,
							CustomID: "finalize_create_standup",
						},
					},
				},
			},
		},
	})
}

func (h *StandupHandler) openSingleAnswerModal(
	session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	standupIDStr string,
	qIndex int) {

	var standupID uint
	fmt.Sscanf(standupIDStr, "%d", &standupID)

	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		log.Println("Error fetching standup for modal:", err)
		return
	}

	if qIndex == 0 {
		state := models.StandupState{
			UserID:    intr.User.ID,
			GuildID:   standup.GuildID,
			StandupID: standup.ID,
			Answers:   []string{},
		}
		redisKey := fmt.Sprintf("%s_%d", intr.User.ID, standup.ID)
		store.SaveState(h.Redis, redisKey, state)
	}

	questionText := standup.Questions[qIndex]

	label := questionText
	if len(label) > 45 {
		label = label[:42] + "..."
	}

	placeholder := questionText
	if len(placeholder) > 100 {
		placeholder = placeholder[:97] + "..."
	}

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("standup_answer_modal_%d_%d", standup.ID, qIndex),
			Title:    fmt.Sprintf("%s (%d/%d)", standup.Name, qIndex+1, len(standup.Questions)),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "answer_text",
							Label:       label,
							Style:       discordgo.TextInputParagraph,
							Required:    true,
							Placeholder: placeholder,
						},
					},
				},
			},
		},
	})

	if err != nil {
		log.Println("Error opening answer modal:", err)
	}
}

func (h *StandupHandler) handleSingleAnswerSubmit(session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	standupID uint,
	qIndex int) {

	answer := intr.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	redisKey := fmt.Sprintf("%s_%d", intr.User.ID, standupID)
	state, err := store.GetState(h.Redis, redisKey)
	if err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Session expired. Please run `/start` to try again.",
				Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	state.Answers = append(state.Answers, strings.TrimSpace(answer))
	store.SaveState(h.Redis, redisKey, *state)

	var standup models.Standup
	h.DB.First(&standup, standupID)

	nextQIndex := qIndex + 1

	if nextQIndex < len(standup.Questions) {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Question %d answered!**\n\nReady for question %d?", qIndex+1, nextQIndex+1),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    fmt.Sprintf("Next: Question %d", nextQIndex+1),
								Style:    discordgo.PrimaryButton,
								CustomID: fmt.Sprintf("continue_standup_%d_%d", standup.ID, nextQIndex),
							},
						},
					},
				},
			},
		})
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "✅ **Standup complete!** Your team has been notified.",
			Components: []discordgo.MessageComponent{},
		},
	})

	h.finalizeStandup(session, state)
	h.Redis.Del(context.Background(), "state:"+redisKey)
}

func (h *StandupHandler) handleSkipStandup(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint) {
	userID := utils.ExtractUserID(intr)

	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		utils.RespondWithError(session, intr.Interaction, "Standup not found.")
		return
	}

	var userProfile models.UserProfile
	h.DB.Where("user_id = ?", userID).First(&userProfile)

	localToday := utils.GetUserLocalTime(userProfile.Timezone).Format("2006-01-02")

	history := models.StandupHistory{
		UserID:    userID,
		StandupID: standupID,
		Date:      localToday,
		Answers:   []string{"Skipped / OOO"},
	}
	h.DB.Create(&history)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("⏭️ %s Update (Skipped)", standup.Name),
		Description: fmt.Sprintf("<@%s> skipped their standup today.", userID),
		Color:       0x808080,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	session.ChannelMessageSendComplex(standup.ReportChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
	})

	utils.UpdateMessage(session, intr,
		"✅ You have successfully skipped today's standup. Your team has been notified!", nil)
}

func (h *StandupHandler) finalizeStandup(s *discordgo.Session, state *models.StandupState) {
	var standup models.Standup
	result := h.DB.First(&standup, state.StandupID)

	if result.Error != nil || standup.ReportChannelID == "" {
		log.Printf("Could not find standup config for ID %d", state.StandupID)
		return
	}

	var userProfile models.UserProfile
	h.DB.Where("user_id = ?", state.UserID).First(&userProfile)

	localToday := utils.GetUserLocalTime(userProfile.Timezone).Format("2006-01-02")

	history := models.StandupHistory{
		UserID:    state.UserID,
		StandupID: state.StandupID,
		Date:      localToday,
		Answers:   state.Answers,
	}

	if err := h.DB.Create(&history).Error; err != nil {
		log.Println("❌ Error saving standup history to database:", err)
	} else {
		log.Printf("✅ Saved history for user %s on %s", state.UserID, localToday)
	}

	var fields []*discordgo.MessageEmbedField
	for i, answer := range state.Answers {
		questionText := "Update"
		if i < len(standup.Questions) {
			questionText = standup.Questions[i]
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   questionText,
			Value:  "👉 " + answer,
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🚀 %s Update", standup.Name),
		Description: fmt.Sprintf("Progress report from <@%s>", state.UserID),
		Color:       0x5865F2,
		// Color:       0x00ff00,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendComplex(standup.ReportChannelID, &discordgo.MessageSend{
		// Content: fmt.Sprintf("🔔 Update from <@%s>", state.UserID),
		Embeds: []*discordgo.MessageEmbed{embed},
	})
}

func (h *StandupHandler) sendStandupSelectionMenu(s *discordgo.Session,
	userID,
	guildID, channelID string,
	standups []models.Standup) {

	targetChannelID := channelID
	if targetChannelID == "" {
		dm, _ := s.UserChannelCreate(userID)
		targetChannelID = dm.ID
	}

	var options []discordgo.SelectMenuOption
	for _, st := range standups {
		if guildID != "" && st.GuildID != guildID {
			continue
		}

		options = append(options, discordgo.SelectMenuOption{
			Label:       st.Name,
			Value:       fmt.Sprintf("%d", st.ID),
			Description: fmt.Sprintf("ID: %d", st.ID),
		})
	}

	if len(options) == 0 {
		s.ChannelMessageSend(targetChannelID, "⛔ You are not part of any standups in this specific server.")
		return
	}

	s.ChannelMessageSendComplex(targetChannelID, &discordgo.MessageSend{
		Content: "Found multiple standups. Please select one:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    "select_standup_join",
						Placeholder: "Choose a standup to join...",
						Options:     options,
					},
				},
			},
		},
	})
}

func (h *StandupHandler) handleStandupSelection(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := utils.ExtractUserID(intr)

	if len(intr.MessageComponentData().Values) == 0 {
		return
	}
	selectedID := intr.MessageComponentData().Values[0]

	var standup models.Standup
	h.DB.First(&standup, selectedID)

	var user models.UserProfile
	if err := h.DB.Preload("Standups").Where("user_id = ?", userID).First(&user).Error; err != nil {
		log.Println("Error finding user:", err)
		return
	}

	h.DB.Model(&user).Association("Standups").Append(&standup)

	utils.UpdateMessage(session, intr, fmt.Sprintf("✅ You joined **%s**!", standup.Name), nil)

	h.InitiateStandup(session, userID, standup.GuildID, intr.ChannelID, standup.ID)
}
