package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

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
	h.DB.Where("user_id = ?", intr.Member.User.ID).FirstOrCreate(&manager,
		models.UserProfile{UserID: intr.Member.User.ID})

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
				timeDisplay := formatLocalTime(standup.Time, user.Timezone)
				welcomeMsg := fmt.Sprintf(
					"👋 **You've been added to the '%s' Standup!**\n\n"+
						"⏰ This standup is scheduled for %s\n\n"+
						"You can now submit your daily reports for this team.\n"+
						"Run `/start` here or in the server to begin.",
					name, timeDisplay,
				)
				session.ChannelMessageSend(dmChannel.ID, welcomeMsg)
			}
		}
	}

	h.Redis.Del(context.Background(), "state:"+intr.Member.User.ID+"_create")

	timeDisplay := formatLocalTime(standup.Time, manager.Timezone)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎉 **Standup '%s' created successfully!**\n⏰ Scheduled for: %s\nAdded %d questions and %d members.",
				standup.Name, timeDisplay, len(standup.Questions), addedCount),
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (h *BotHanlder) handleEditStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	userID := extractUserID(intr)

	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}

	standupName := optionMap["standup_name"].StringValue()

	var standup models.Standup
	result := h.DB.Where("guild_id = ? AND name = ?", intr.GuildID, standupName).First(&standup)
	if result.Error != nil {
		respondWithError(session, intr.Interaction,
			fmt.Sprintf("Standup named **%s** not found in this server.", standupName))
		return
	}

	if standup.ManagerID != userID && !isServerAdmin(intr) {
		respondWithError(session, intr.Interaction,
			"⛔ Only the manager who created this standup, or a Server Admin, can edit it.")
		return
	}

	updatedFields := make([]string, 0)

	if opt, ok := optionMap["new_channel"]; ok {
		standup.ReportChannelID = opt.ChannelValue(session).ID
		updatedFields = append(updatedFields, fmt.Sprintf("Report Channel (<#%s>)", standup.ReportChannelID))
	}

	if opt, ok := optionMap["new_time"]; ok {
		newTime := opt.StringValue()
		var hTime, mTime int
		if _, err := fmt.Sscanf(newTime, "%d:%d", &hTime, &mTime); err != nil || hTime < 0 || hTime > 23 || mTime < 0 || mTime > 59 {
			respondWithError(session, intr.Interaction,
				"⛔ Invalid time format. Please use HH:MM in 24h format (e.g., 09:30).")
			return
		}
		standup.Time = fmt.Sprintf("%02d:%02d", hTime, mTime)
		updatedFields = append(updatedFields, fmt.Sprintf("Trigger Time (%s)", standup.Time))
	}

	responseMsg := fmt.Sprintf("⚙️ **Managing %s**\n", standup.Name)
	if len(updatedFields) > 0 {
		h.DB.Save(&standup)
		responseMsg += fmt.Sprintf("✅ *Saved changes to:*\n- %s\n\n", strings.Join(updatedFields, "\n- "))
	} else {
		responseMsg += "ℹ️ No basic settings were changed.\n\n"
	}
	responseMsg += "Click the button below to edit your team's daily questions!"

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseMsg,
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "📝 Edit Questions",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("open_q_dash_%d", standup.ID),
						},
					},
				},
			},
		},
	})
}

func (h *BotHanlder) showQuestionDashboard(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint, isUpdate bool) {

	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		respondWithError(session, intr.Interaction, "Standup not found.")
		return
	}

	var qList strings.Builder
	var options []discordgo.SelectMenuOption

	for i, q := range standup.Questions {
		qList.WriteString(fmt.Sprintf("**%d.** %s\n", i+1, q))

		label := q
		if len(label) > 90 {
			label = label[:87] + "..."
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       fmt.Sprintf("Edit Question %d", i+1),
			Description: label,
			Value:       fmt.Sprintf("%d", i),
		})
	}

	if len(standup.Questions) == 0 {
		qList.WriteString("*No questions yet! Add one below.*\n")
	}

	var components []discordgo.MessageComponent

	if len(options) > 0 {
		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("select_q_%d", standup.ID),
					Placeholder: "Select a question to edit or delete...",
					Options:     options,
				},
			},
		})
	}

	components = append(components, discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "➕ Add New Question",
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("add_q_btn_%d", standup.ID),
			},
			discordgo.Button{
				Label:    "✅ Finish & Save",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("finish_q_dash_%d", standup.ID),
			},
		},
	})

	content := fmt.Sprintf("📋 **Managing Questions for %s**\n\n%s\n*💡 To delete a question, select it and completely clear the text box!*",
		standup.Name, qList.String())

	respType := discordgo.InteractionResponseChannelMessageWithSource
	if isUpdate {
		respType = discordgo.InteractionResponseUpdateMessage
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: respType,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *BotHanlder) handleEditSingleQuestionPrompt(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint, qIndex int) {

	var standup models.Standup
	h.DB.First(&standup, standupID)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("edit_single_q_%d_%d", standup.ID, qIndex),
			Title:    fmt.Sprintf("Edit Question %d", qIndex+1),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "q_text",
							Label:     "Question Text (Clear to Delete)",
							Style:     discordgo.TextInputParagraph,
							Value:     standup.Questions[qIndex],
							Required:  false,
							MaxLength: 300,
						},
					},
				},
			},
		},
	})
}

func (h *BotHanlder) handleAddQuestionPrompt(session *discordgo.Session, intr *discordgo.InteractionCreate,
	standupID uint) {
	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("add_single_q_%d", standupID),
			Title:    "Add New Question",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:  "q_text",
							Label:     "Type your new question",
							Style:     discordgo.TextInputParagraph,
							Required:  true,
							MaxLength: 300,
						},
					},
				},
			},
		},
	})
}

func (h *BotHanlder) handleQuestionSubmit(session *discordgo.Session, intr *discordgo.InteractionCreate,
	standupID uint, qIndex int, isNew bool) {

	var standup models.Standup
	h.DB.First(&standup, standupID)

	newText := strings.TrimSpace(intr.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value)

	if isNew {
		standup.Questions = append(standup.Questions, newText)
	} else {
		if newText == "" {
			standup.Questions = append(standup.Questions[:qIndex], standup.Questions[qIndex+1:]...)
		} else {
			standup.Questions[qIndex] = newText
		}
	}

	h.DB.Save(&standup)
	h.showQuestionDashboard(session, intr, standup.ID, true)
}

func (h *BotHanlder) handleFinishQuestionEdit(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint) {

	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		respondWithError(session, intr.Interaction, "Standup not found.")
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ **Done!** The questions for **%s** are locked in.\n*(Total questions: %d)*",
				standup.Name, len(standup.Questions)),
			Components: []discordgo.MessageComponent{},
		},
	})
}

func (h *BotHanlder) handleDeleteStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	standupName := intr.ApplicationCommandData().Options[0].StringValue()
	userID := extractUserID(intr)

	var standup models.Standup
	result := h.DB.Preload("Participants").
		Where("guild_id = ? and name = ?", intr.GuildID, standupName).
		First(&standup)
	if result.Error != nil {
		respondWithError(session, intr.Interaction, fmt.Sprintf("Standup named **%s** not found in this server.", standupName))
		return
	}

	if standup.ManagerID != userID && !isServerAdmin(intr) {
		respondWithError(session, intr.Interaction, "⛔ Only the manager who created this standup can delete it.")
		return
	}

	if err := h.DB.Model(&standup).Association("Participants").Clear(); err != nil {
		log.Println("Error clearing standup participants during deletion:", err)
	}

	if err := h.DB.Unscoped().Delete(&standup).Error; err != nil {
		respondWithError(session, intr.Interaction, "Failed to delete the standup from the database.")
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🗑️ ✅ Standup **%s** and all its participant links have been permanently deleted.", standup.Name),
		},
	})
}

func (h *BotHanlder) handleSetChannel(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetChannelID := options[0].Value.(string)
	standupName := options[1].Value.(string)

	var standup models.Standup
	result := h.DB.Where("guild_id = ? AND name = ?", intr.GuildID, standupName).First(&standup)

	if result.Error != nil {
		respondWithError(session, intr.Interaction, "Standup not found. Create it first with `/create-standup`.")
		return
	}

	standup.ReportChannelID = targetChannelID
	h.DB.Save(&standup)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Reports for **%s** will now be sent to <#%s>", standup.Name, targetChannelID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *BotHanlder) handleAddMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetUser := options[0].UserValue(session)
	targetStandupName := options[1].StringValue()
	userID := extractUserID(intr)

	var standup models.Standup
	result := h.DB.Where("guild_id = ? and name = ?", intr.GuildID, targetStandupName).First(&standup)
	if result.Error != nil {
		respondWithError(session, intr.Interaction,
			fmt.Sprintf("Standup named **%s** not found in this server.", targetStandupName))
		return
	}

	if standup.ManagerID != userID && !isServerAdmin(intr) {
		respondWithError(session, intr.Interaction,
			"⛔ Only the manager who created this standup, or a Server Admin, can edit it.")
		return
	}

	var targetProfile models.UserProfile
	h.DB.Preload("Standups", "id = ?", standup.ID).Where("user_id = ?", targetUser.ID).
		FirstOrCreate(&targetProfile, models.UserProfile{UserID: targetUser.ID})

	if len(targetProfile.Standups) > 0 {
		respondWithError(session, intr.Interaction, fmt.Sprintf("⛔ <@%s> is already a member of **%s**.",
			targetUser.ID, standup.Name))
		return
	}

	if err := h.StandupService.AddMemberToStandup(targetUser.ID, standup.ID); err != nil {
		respondWithError(session, intr.Interaction, "Failed to add member.")
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
		timeDisplay := formatLocalTime(standup.Time, targetProfile.Timezone)
		welcomeMsg := fmt.Sprintf(
			"👋 **You've been added to the '%s' Standup!**\n\n"+
				"⏰ This standup is scheduled for %s\n\n"+
				"You can now submit your daily reports for this team.\n"+
				"Run `/start` here or in the server to begin.",
			standup.Name, timeDisplay,
		)
		session.ChannelMessageSend(dmChannel.ID, welcomeMsg)
	}
}

func (h *BotHanlder) handleRemoveMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetUser := options[0].UserValue(session)
	targetStandupName := options[1].StringValue()
	userID := extractUserID(intr)

	var standup models.Standup
	result := h.DB.Where("guild_id = ? and name = ?", intr.GuildID, targetStandupName).First(&standup)
	if result.Error != nil {
		respondWithError(session, intr.Interaction, fmt.Sprintf("Standup named **%s** not found in this server.",
			targetStandupName))
		return
	}

	if standup.ManagerID != userID && !isServerAdmin(intr) {
		respondWithError(session, intr.Interaction, "⛔ Only the manager who created this standup, or a Server Admin, can edit it.")
		return
	}

	var targetProfile models.UserProfile
	h.DB.Preload("Standups", "id = ?", standup.ID).Where("user_id = ?", targetUser.ID).First(&targetProfile)

	if targetProfile.ID == 0 || len(targetProfile.Standups) == 0 {
		respondWithError(session, intr.Interaction,
			fmt.Sprintf("⛔ <@%s> is not currently a member of **%s**.", targetUser.ID, standup.Name))
		return
	}

	if err := h.StandupService.RemoveMemberFromStandup(targetUser.ID, standup.ID); err != nil {
		respondWithError(session, intr.Interaction, "Failed to remove member.")
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
