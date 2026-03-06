package standup

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Gurkunwar/asyncflow/internal/bot/utils"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (h *StandupHandler) fetchAuthorizedStandup(session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	standupName string) (*models.Standup, bool) {

	var standup models.Standup
	if err := h.DB.Where("guild_id = ? AND name = ?", intr.GuildID, standupName).
		First(&standup).Error; err != nil {
		utils.RespondWithError(session, intr.Interaction,
			fmt.Sprintf("Standup named **%s** not found in this server.", standupName))
		return nil, false
	}

	userID := utils.ExtractUserID(intr)
	if standup.ManagerID != userID && !utils.IsServerAdmin(intr) {
		utils.RespondWithError(session, intr.Interaction,
			"⛔ Only the manager who created this standup, or a Server Admin, can manage it.")
		return nil, false
	}

	return &standup, true
}

func generateDaysMenu(standupID uint, activeDaysStr string) discordgo.MessageComponent {
	if activeDaysStr == "" {
		activeDaysStr = "Monday,Tuesday,Wednesday,Thursday,Friday"
	}

	daysMap := make(map[string]bool)
	for _, d := range strings.Split(activeDaysStr, ",") {
		daysMap[strings.TrimSpace(d)] = true
	}

	daysOfWeek := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	var dayOptions []discordgo.SelectMenuOption
	for _, d := range daysOfWeek {
		dayOptions = append(dayOptions, discordgo.SelectMenuOption{
			Label:   d,
			Value:   d,
			Default: daysMap[d],
		})
	}

	minValues := 1
	return discordgo.SelectMenu{
		CustomID:    fmt.Sprintf("edit_days_%d", standupID),
		Placeholder: "Select active days...",
		MinValues:   &minValues,
		MaxValues:   7,
		Options:     dayOptions,
	}
}

func (h *StandupHandler) handleCreateStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
    userID := utils.ExtractUserID(intr)
    optMap := utils.ParseCommandOptions(intr)

    name := optMap["name"].StringValue()
    channelID := optMap["channel"].ChannelValue(session).ID
    membersRaw := optMap["members"].StringValue()

    standupTime := "09:00"
    if opt, ok := optMap["time"]; ok {
        standupTime = opt.StringValue()
    }

    defaultQuestions := []string{
        "What did you accomplish yesterday?",
        "What will you do today?",
        "Are you stuck anywhere? (Blockers)",
    }

    standupInput := models.Standup{
        Name:            name,
        ReportChannelID: channelID,
        GuildID:         intr.GuildID,
        ManagerID:       userID,
        Time:            standupTime,
        Questions:       defaultQuestions,
        Days:            "Monday,Tuesday,Wednesday,Thursday,Friday",
    }

    var manager models.UserProfile
    h.DB.FirstOrCreate(&manager, models.UserProfile{UserID: userID})

    createdStandup, err := h.StandupService.CreateStandup(standupInput)
    if err != nil {
        utils.RespondWithMessage(session, intr, "❌ Failed to create standup.", true)
        return
    }

    addedCount := 0
    re := regexp.MustCompile(`<@!?(\d+)>`)
    matches := re.FindAllStringSubmatch(membersRaw, -1)

    for _, match := range matches {
        if len(match) > 1 {
            targetUserID := match[1]
            var user models.UserProfile
            h.DB.FirstOrCreate(&user, models.UserProfile{UserID: targetUserID})
            
            h.DB.Model(&user).Association("Standups").Append(createdStandup)
            addedCount++

            go func(uID string, tz string) {
                dmChannel, err := session.UserChannelCreate(uID)
                if err == nil {
                    timeDisplay := utils.FormatLocalTime(createdStandup.Time, tz)
                    welcomeMsg := fmt.Sprintf("👋 **You've been added to the '%s' Standup!**\n\n"+
                        "⏰ Scheduled for: %s\nRun `/start` here or in the server to begin.",
                        createdStandup.Name, timeDisplay)
                    session.ChannelMessageSend(dmChannel.ID, welcomeMsg)
                }
            }(targetUserID, user.Timezone)
        }
    }

    timeDisplay := utils.FormatLocalTime(createdStandup.Time, manager.Timezone)
    successMsg := fmt.Sprintf("🎉 **Standup '%s' created successfully!**\n"+
        "⏰ Scheduled for: **%s** on **Monday-Friday**\n"+
        "👥 Added **%d** members.\n\n"+
        "💡 *I have assigned the standard 3 Agile questions. Use `/edit-standup` "+
        "to customize your questions or active days!*",
        createdStandup.Name, timeDisplay, addedCount)

    utils.RespondWithMessage(session, intr, successMsg, true)
}

func (h *StandupHandler) handleEditStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	optMap := utils.ParseCommandOptions(intr)
	standupName := optMap["standup_name"].StringValue()

	standup, authorized := h.fetchAuthorizedStandup(session, intr, standupName)
	if !authorized {
		return
	}

	updatedFields := make([]string, 0)

	if opt, ok := optMap["new_channel"]; ok {
		standup.ReportChannelID = opt.ChannelValue(session).ID
		updatedFields = append(updatedFields, fmt.Sprintf("Report Channel (<#%s>)", standup.ReportChannelID))
	}

	if opt, ok := optMap["new_time"]; ok {
		newTime := opt.StringValue()
		var hTime, mTime int
		if _, err := fmt.Sscanf(newTime, "%d:%d", &hTime, &mTime); err != nil ||
			hTime < 0 || hTime > 23 || mTime < 0 || mTime > 59 {
			utils.RespondWithError(session, intr.Interaction,
				"⛔ Invalid time format. Please use HH:MM in 24h format (e.g., 09:30).")
			return
		}
		standup.Time = fmt.Sprintf("%02d:%02d", hTime, mTime)
		updatedFields = append(updatedFields, fmt.Sprintf("Trigger Time (%s)", standup.Time))
	}

	responseMsg := fmt.Sprintf("⚙️ **Managing %s**\n", standup.Name)
	if len(updatedFields) > 0 {
		h.DB.Save(standup)
		responseMsg += fmt.Sprintf("✅ *Saved changes to:*\n- %s\n\n", strings.Join(updatedFields, "\n- "))
	} else {
		responseMsg += "ℹ️ No basic settings were changed.\n\n"
	}
	responseMsg += "Use the menu below to change active days, or edit your team's questions!"

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: responseMsg,
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						generateDaysMenu(standup.ID, standup.Days),
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "📝 Edit Questions",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("open_q_dash_%d", standup.ID),
						},
						discordgo.Button{
							Label:    "✅ Done",
							Style:    discordgo.SuccessButton,
							CustomID: fmt.Sprintf("finish_q_dash_%d", standup.ID),
						},
					},
				},
			},
		},
	})
}

func (h *StandupHandler) handleEditDaysSubmit(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint) {
	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		utils.RespondWithError(session, intr.Interaction, "Standup not found.")
		return
	}

	selectedDays := intr.MessageComponentData().Values
	standup.Days = strings.Join(selectedDays, ",")
	h.DB.Save(&standup)

	prettyDays := strings.ReplaceAll(standup.Days, ",", ", ")

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Active days for **%s** have been updated to:\n**%s**\n\n"+
				"Use the menu below to make further changes, or click Done.", standup.Name, prettyDays),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						generateDaysMenu(standup.ID, standup.Days),
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "📝 Edit Questions",
							Style:    discordgo.SecondaryButton,
							CustomID: fmt.Sprintf("open_q_dash_%d", standup.ID),
						},
						discordgo.Button{
							Label:    "✅ Done",
							Style:    discordgo.SuccessButton,
							CustomID: fmt.Sprintf("finish_q_dash_%d", standup.ID),
						},
					},
				},
			},
		},
	})
}

func (h *StandupHandler) handleDeleteStandup(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	optMap := utils.ParseCommandOptions(intr)

	targetName := ""
	if opt, ok := optMap["name"]; ok {
		targetName = opt.StringValue()
	}
	if opt, ok := optMap["standup_name"]; ok {
		targetName = opt.StringValue()
	}

	standup, authorized := h.fetchAuthorizedStandup(session, intr, targetName)
	if !authorized {
		return
	}

	if err := h.StandupService.DeleteStandup(standup.ID); err != nil {
		utils.RespondWithError(session, intr.Interaction, "Failed to delete standup.")
		return
	}

	utils.RespondWithMessage(session, intr, fmt.Sprintf("🗑️ ✅ Standup **%s** deleted.", standup.Name), true)
}

func (h *StandupHandler) handleAddMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	optMap := utils.ParseCommandOptions(intr)
	targetUser := optMap["user"].UserValue(session)
	standupName := optMap["standup_name"].StringValue()

	standup, authorized := h.fetchAuthorizedStandup(session, intr, standupName)
	if !authorized {
		return
	}

	var targetProfile models.UserProfile
	h.DB.Preload("Standups", "id = ?", standup.ID).Where("user_id = ?", targetUser.ID).
		FirstOrCreate(&targetProfile, models.UserProfile{UserID: targetUser.ID})

	if len(targetProfile.Standups) > 0 {
		utils.RespondWithError(session, intr.Interaction,
			fmt.Sprintf("⛔ <@%s> is already a member of **%s**.", targetUser.ID, standup.Name))
		return
	}

	if err := h.StandupService.AddMemberToStandup(targetUser.ID, standup.ID); err != nil {
		utils.RespondWithError(session, intr.Interaction, "Failed to add member.")
		return
	}

	utils.RespondWithMessage(session, intr, fmt.Sprintf("✅ <@%s> has been added to **%s**!",
		targetUser.ID, standup.Name), true)
}

func (h *StandupHandler) handleRemoveMember(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	optMap := utils.ParseCommandOptions(intr)
	targetUser := optMap["user"].UserValue(session)
	standupName := optMap["standup_name"].StringValue()

	standup, authorized := h.fetchAuthorizedStandup(session, intr, standupName)
	if !authorized {
		return
	}

	var targetProfile models.UserProfile
	h.DB.Preload("Standups", "id = ?", standup.ID).Where("user_id = ?", targetUser.ID).First(&targetProfile)

	if targetProfile.ID == 0 || len(targetProfile.Standups) == 0 {
		utils.RespondWithError(session, intr.Interaction,
			fmt.Sprintf("⛔ <@%s> is not currently a member of **%s**.", targetUser.ID, standup.Name))
		return
	}

	if err := h.StandupService.RemoveMemberFromStandup(targetUser.ID, standup.ID); err != nil {
		utils.RespondWithError(session, intr.Interaction, "Failed to remove member.")
		return
	}

	utils.RespondWithMessage(session, intr, fmt.Sprintf("✅ <@%s> has been successfully removed from **%s**.",
		targetUser.ID, standup.Name), true)
}

func (h *StandupHandler) handleStandupInfo(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	optMap := utils.ParseCommandOptions(intr)
	standupName := optMap["standup_name"].StringValue()

	var standup models.Standup
	if err := h.DB.Preload("Participants").Where("guild_id = ? AND name = ?", intr.GuildID, standupName).
		First(&standup).Error; err != nil {
		utils.RespondWithError(session, intr.Interaction,
			fmt.Sprintf("❌ Standup named **%s** not found.", standupName))
		return
	}

	activeDays := standup.Days
	if activeDays == "" {
		activeDays = "Monday, Tuesday, Wednesday, Thursday, Friday"
	} else {
		activeDays = strings.ReplaceAll(activeDays, ",", ", ")
	}

	var qList strings.Builder
	for i, q := range standup.Questions {
		qList.WriteString(fmt.Sprintf("**%d.** %s\n", i+1, q))
	}
	if len(standup.Questions) == 0 {
		qList.WriteString("*No questions configured.*")
	}

	var memberMentions []string
	for _, p := range standup.Participants {
		memberMentions = append(memberMentions, fmt.Sprintf("<@%s>", p.UserID))
	}
	memberStr := strings.Join(memberMentions, " ")

	if len(memberStr) > 1000 {
		memberStr = fmt.Sprintf("*%d members (List too long to display)*", len(standup.Participants))
	} else if len(memberMentions) == 0 {
		memberStr = "*No members added yet.*"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📊 Standup Info: %s", standup.Name),
		Color:       0x5865F2,
		Description: "Here is the current configuration for this team.",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "👑 Manager", Value: fmt.Sprintf("<@%s>", standup.ManagerID), Inline: true},
			{Name: "📢 Report Channel", Value: fmt.Sprintf("<#%s>", standup.ReportChannelID), Inline: true},
			{Name: "⏰ Trigger Time", Value: fmt.Sprintf("**%s** (Local to each user)", standup.Time), Inline: true},
			{Name: "📅 Active Days", Value: activeDays, Inline: false},
			{Name: fmt.Sprintf("👥 Members (%d)", len(standup.Participants)), Value: memberStr, Inline: false},
			{Name: "📝 Questions", Value: qList.String(), Inline: false},
		},
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *StandupHandler) showQuestionDashboard(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint, isUpdate bool) {
	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		utils.RespondWithError(session, intr.Interaction, "Standup not found.")
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
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("add_q_btn_%d", standup.ID),
			},
			discordgo.Button{
				Label:    "✅ Done",
				Style:    discordgo.SuccessButton,
				CustomID: fmt.Sprintf("finish_q_dash_%d", standup.ID),
			},
		},
	})

	content := fmt.Sprintf("📋 **Managing Questions for %s**\n\n%s\n*💡 To delete a question, "+
		"select it and completely clear the text box!*", standup.Name, qList.String())

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

func (h *StandupHandler) handleEditSingleQuestionPrompt(session *discordgo.Session,
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

func (h *StandupHandler) handleAddQuestionPrompt(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint) {
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

func (h *StandupHandler) handleQuestionSubmit(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint, qIndex int, isNew bool) {
	var standup models.Standup
	h.DB.First(&standup, standupID)

	newText := strings.TrimSpace(intr.ModalSubmitData().
		Components[0].(*discordgo.ActionsRow).
		Components[0].(*discordgo.TextInput).Value)

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

func (h *StandupHandler) handleFinishQuestionEdit(session *discordgo.Session,
	intr *discordgo.InteractionCreate, standupID uint) {
	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		utils.RespondWithError(session, intr.Interaction, "Standup not found.")
		return
	}

	utils.UpdateMessage(session, intr,
		fmt.Sprintf("✅ **Done!** The settings and questions for **%s** are fully saved and locked in.",
			standup.Name), nil)
}
