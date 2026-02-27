package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) InitiateStandup(s *discordgo.Session, userID string, guildID, channelID string, standupID uint) {
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

func (h *BotHanlder) finalizeStandup(s *discordgo.Session, state *models.StandupState) {
	var standup models.Standup
	result := h.DB.First(&standup, state.StandupID)

	if result.Error != nil || standup.ReportChannelID == "" {
		log.Printf("Could not find standup config for ID %d", state.StandupID)
		return
	}

	var userProfile models.UserProfile
	h.DB.Where("user_id = ?", state.UserID).First(&userProfile)

	loc, err := time.LoadLocation(userProfile.Timezone)
	if err != nil {
		loc = time.UTC
	}
	localToday := time.Now().In(loc).Format("2006-01-02")

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

	s.ChannelMessageSendEmbed(standup.ReportChannelID, embed)
}

func (h *BotHanlder) handleCreateStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	name := options[0].StringValue()
	channelID := options[1].ChannelValue(session).ID
	membersRaw := options[2].StringValue()
	standupTime := "09:00"
	if len(options) > 3 {
		standupTime = options[3].StringValue()
	}

	tempState := models.StandupState{
		UserID:  intr.Member.User.ID,
		GuildID: intr.GuildID,
		Answers: []string{name, channelID, membersRaw, standupTime},
	}

	store.SaveState(h.Redis, intr.Member.User.ID+"_create", tempState)
	h.openSingleQuestionModal(session, intr, 1)
}

func (h *BotHanlder) openSingleQuestionModal(session *discordgo.Session, intr *discordgo.InteractionCreate, qNum int) {
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

func (h *BotHanlder) handleCreateQuestionSubmit(session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	customID string) {

	var currentQNum int
	fmt.Sscanf(customID, "create_q_modal_%d", &currentQNum)

	newQuestion := intr.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

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

func (h *BotHanlder) finalizeCreateStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	state, err := store.GetState(h.Redis, intr.Member.User.ID+"_create")
	if err != nil || len(state.Answers) < 5 {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Session expired or no questions added.", 
			Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	name := state.Answers[0]
	channelID := state.Answers[1]
	membersRaw := state.Answers[2]
	standupTime := state.Answers[3]
	questions := state.Answers[4:]

	standup := models.Standup{
		Name:            name,
		ReportChannelID: channelID,
		GuildID:         intr.GuildID,
		ManagerID:       intr.Member.User.ID,
		Time:            standupTime,
		Questions:       questions,
	}

	var guild models.Guild
	h.DB.Where("id = ?", intr.GuildID).FirstOrCreate(&guild, models.Guild{GuildID: intr.GuildID})

	var manager models.UserProfile
	h.DB.Where("user_id = ?", intr.Member.User.ID).FirstOrCreate(&manager, models.UserProfile{UserID: intr.Member.User.ID})

	if err := h.StandupService.CreateStandup(standup); err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Failed to create standup."},
		})
		return
	}

	members := strings.Fields(membersRaw)
	addedCount := 0

	h.DB.Where("guild_id = ? AND name = ?", intr.GuildID, name).First(&standup)

	for _, member := range members {
		if strings.HasPrefix(member, "<@") && strings.HasSuffix(member, ">") {
			userID := strings.Trim(member, "<@!>")

			var user models.UserProfile
			h.DB.FirstOrCreate(&user, models.UserProfile{UserID: userID})
			h.DB.Model(&user).Association("Standups").Append(&standup)
			addedCount++

			dmChannel, err := session.UserChannelCreate(userID)
			if err == nil {
				welcomeMsg := fmt.Sprintf(
					"👋 **You've been added to the '%s' Standup!**\n\n"+
						"You can now submit your daily reports for this team.\n"+
						"Run `/start` here or in the server to begin.",
					name,
				)
				session.ChannelMessageSend(dmChannel.ID, welcomeMsg)
			}
		}
	}

	h.Redis.Del(context.Background(), "state:"+intr.Member.User.ID+"_create")

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎉 **Team '%s' created successfully!**\nAdded %d questions and %d members.",
				standup.Name, len(standup.Questions), addedCount),
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (h *BotHanlder) handleEditStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options

	var userID string
	if intr.Member != nil {
		userID = intr.Member.User.ID
	} else {
		userID = intr.User.ID
	}

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	standupName := optionMap["standup_name"].StringValue()

	var standup models.Standup
	result := h.DB.Where("guild_id = ? AND name = ?", intr.GuildID, standupName).First(&standup)
	if result.Error != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Standup named **%s** not found in this server.", standupName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if standup.ManagerID != userID {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ Only the manager who created this standup can edit it.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	updatedFields := make([]string, 0)

	if opt, ok := optionMap["new_channel"]; ok {
		standup.ReportChannelID = opt.ChannelValue(session).ID
		updatedFields = append(updatedFields, "Report Channel")
	}

	if opt, ok := optionMap["new_questions"]; ok {
		questionsRaw := opt.StringValue()
		rawList := strings.Split(questionsRaw, ";")
		var cleanQuestions []string
		for _, q := range rawList {
			cleanQ := strings.TrimSpace(q)
			if cleanQ != "" {
				cleanQuestions = append(cleanQuestions, cleanQ)
			}
		}
		if len(cleanQuestions) > 0 {
			standup.Questions = cleanQuestions
			updatedFields = append(updatedFields, "Questions")
		}
	}

	if opt, ok := optionMap["new_time"]; ok {
		standup.Time = opt.StringValue()
		updatedFields = append(updatedFields, "Trigger Time")
	}

	if len(updatedFields) == 0 {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ℹ️ No changes were provided to update.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := h.DB.Save(&standup).Error; err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Failed to save updates to the database.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	responseMsg := fmt.Sprintf("✅ **%s** has been successfully updated!\n*Changes made to:* %s",
		standup.Name,
		strings.Join(updatedFields, ", "))

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseMsg,
		},
	})
}

func (h *BotHanlder) handleDeleteStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	standupName := intr.ApplicationCommandData().Options[0].StringValue()

	var userID string
	if intr.Member != nil {
		userID = intr.Member.User.ID
	} else {
		userID = intr.User.ID
	}

	var standup models.Standup
	result := h.DB.Preload("Participants").
		Where("guild_id = ? and name = ?", intr.GuildID, standupName).
		First(&standup)
	if result.Error != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Standup named **%s** not found in this server.", standupName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if standup.ManagerID != userID {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ Only the manager who created this standup can delete it.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := h.DB.Model(&standup).Association("Participants").Clear(); err != nil {
		log.Println("Error clearing standup participants during deletion:", err)
	}

	if err := h.DB.Unscoped().Delete(&standup).Error; err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Failed to delete the standup from the database.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🗑️ ✅ Standup **%s** and all its participant links have been permanently deleted.",
				standup.Name),
		},
	})
}

func (h *BotHanlder) handleAddMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetUser := options[0].UserValue(session)
	targetStandupName := options[1].StringValue()

	var standup models.Standup
	result := h.DB.Where("guild_id = ? and name = ?", intr.GuildID, targetStandupName).First(&standup)
	if result.Error != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Standup named **%s** not found in this server.", targetStandupName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if standup.ManagerID != intr.Member.User.ID {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ You are not the manager of this standup.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := h.StandupService.AddMemberToStandup(targetUser.ID, standup.ID); err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Failed to add member.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ <@%s> has been added to **%s**!", targetUser.ID, standup.Name),
		},
	})

	dmChannel, err := session.UserChannelCreate(targetUser.ID)
	if err == nil {
		session.ChannelMessageSend(dmChannel.ID, fmt.Sprintf(
			`👋 You've been added to the **%s** standup by your manager.
			\nRun "/start" in the server to submit your daily report.`,
			standup.Name))
	}
}

func (h *BotHanlder) handleRemoveMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetUser := options[0].UserValue(session)
	targetStandup := options[1].StringValue()

	var standup models.Standup
	result := h.DB.Where("guild_id = ? and name = ?", intr.GuildID, targetStandup).First(&standup)
	if result.Error != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Standup named **%s** not found in this server.", targetStandup),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if standup.ManagerID != intr.Member.User.ID {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ You are not the manager of this standup.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := h.StandupService.RemoveMemberFromStandup(targetUser.ID, standup.ID); err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Failed to remove member.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ <@%s> has been successfully removed from **%s**.", targetUser.ID, standup.Name),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	dmChannel, err := session.UserChannelCreate(targetUser.ID)
	if err == nil {
		session.ChannelMessageSend(dmChannel.ID, fmt.Sprintf(
			"ℹ️ You have been removed from the **%s** standup team by the manager.",
			standup.Name))
	}
}

func (h *BotHanlder) handleHistory(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetUser := options[0].UserValue(session)
	standupName := options[1].StringValue()
	days := 5

	if len(options) > 2 {
		days = int(options[2].IntValue())
		if days > 10 {
			days = 10
		}
	}

	var callerID string
	if intr.Member != nil {
		callerID = intr.Member.User.ID
	} else {
		callerID = intr.User.ID
	}

	var standup models.Standup
	if err := h.DB.
		Where("guild_id = ? and name = ?", intr.GuildID, standupName).
		First(&standup).
		Error; err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("❌ Standup named **%s** not found.", standupName),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if standup.ManagerID != callerID && targetUser.ID != callerID {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "⛔ You can only view your own history, or history for teams you manage.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	cutoffDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	var histories []models.StandupHistory

	h.DB.Where("user_id = ? AND standup_id = ? AND date >= ?", targetUser.ID, standup.ID, cutoffDate).
		Order("date desc").
		Limit(10).
		Find(&histories)

	if len(histories) == 0 {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("📭 No standup history found for <@%s> in **%s** over the last %d days.",
					targetUser.ID, standup.Name, days),
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var embeds []*discordgo.MessageEmbed
	for _, hist := range histories {
		var fields []*discordgo.MessageEmbedField

		for i, answer := range hist.Answers {
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

		embeds = append(embeds, &discordgo.MessageEmbed{
			Title:  fmt.Sprintf("📅 Report from %s", hist.Date),
			Color:  0x5865F2,
			Fields: fields,
		})
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("📜 **Standup History for <@%s> in %s**", targetUser.ID, standup.Name),
			Embeds:  embeds,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *BotHanlder) sendTimezoneMenu(session *discordgo.Session, channelID, userID string, standupID uint) {
	state := models.StandupState{
		UserID:    userID,
		StandupID: standupID,
	}
	store.SaveState(h.Redis, userID, state)

	options := []discordgo.SelectMenuOption{
		{Label: "India (IST)", Value: "Asia/Kolkata", Description: "UTC+5:30"},
		{Label: "US East (EST)", Value: "America/New_York", Description: "UTC-5:00"},
		{Label: "London (GMT)", Value: "Europe/London", Description: "UTC+0:00"},
		{Label: "Singapore (SGT)", Value: "Asia/Singapore", Description: "UTC+8:00"},
	}

	session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "Welcome to **DailyBot**! I don't know your timezone yet. Please pick one:",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.SelectMenu{
						CustomID:    "select_tz",
						Placeholder: "Select your local timezone",
						Options:     options,
					},
				},
			},
		},
	})
}

func (h *BotHanlder) startQuestionFlow(session *discordgo.Session, channelID, userID string, standup models.Standup) {
	state := models.StandupState{
		UserID:    userID,
		GuildID:   standup.GuildID,
		StandupID: standup.ID,
		Answers:   []string{},
	}
	store.SaveState(h.Redis, userID, state)

	session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "Ready to submit your daily standup?",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Fill Daily Standup",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("open_standup_modal_%d", standup.ID),
					},
				},
			},
		},
	})
}

func (h *BotHanlder) sendStandupSelectionMenu(s *discordgo.Session,
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

func (h *BotHanlder) handleStandupSelection(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	var userID string
	if intr.Member != nil {
		userID = intr.Member.User.ID
	} else {
		userID = intr.User.ID
	}

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

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("✅ You joined **%s**!", standup.Name),
			Components: []discordgo.MessageComponent{},
		},
	})

	h.InitiateStandup(session, userID, standup.GuildID, intr.ChannelID, standup.ID)
}
