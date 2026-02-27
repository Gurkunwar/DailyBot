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
	userID := extractUserID(intr)

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
	userID := extractUserID(intr)

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
			Flags: discordgo.MessageFlagsEphemeral,
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
					"👋 **You've been added to the '%s' Standup!**\n\n"+
						"You can now submit your daily reports for this team.\n"+
						"Run `/start` here or in the server to begin.",
					standup.Name,
				))
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