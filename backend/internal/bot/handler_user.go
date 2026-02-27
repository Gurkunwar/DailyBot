package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) handleHelp(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	helpText := "💡 **DailyBot Help Menu**\n\n" +
		"**👤 User Commands**\n" +
		"`/start` - Manually trigger your daily standup form.\n" +
		"`/history` - View past standup reports.\n" +
		"`/timezone` - Set your local timezone for standup reminders.\n" +
		"`/delete-my-data` - Permanently delete your profile and leave all standups.\n\n" +
		"**🛠️ Manager Commands (Admin Only)**\n" +
		"`/create-standup` - Create a new team standup.\n" +
		"`/edit-standup` - Edit an existing standup team's settings.\n" +
		"`/delete-standup` - Permanently delete an existing standup team.\n" +
		"`/set-channel` - Set or change where reports are posted.\n" +
		"`/add-member` - Add a user to an existing standup.\n" +
		"`/remove-member` - Remove a user from an existing standup.\n\n" +
		"ℹ️ *Note: I will automatically ping you at your standup's scheduled time in your saved timezone!*"

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: helpText,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *BotHanlder) handleDeleteMyData(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := extractUserID(intr)

	var user models.UserProfile
	if err := h.DB.Unscoped().Where("user_id = ?", userID).First(&user).Error; err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ No profile found to reset.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if err := h.DB.Model(&user).Association("Standups").Clear(); err != nil {
		log.Println("Error clearing standup teams:", err)
	}

	if result := h.DB.Unscoped().Delete(&user); result.Error != nil {
		session.ChannelMessageSend(intr.ChannelID, "❌ Failed to reset profile.")
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ **Profile Reset Complete.** You have been removed from all standup teams.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
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

func (h *BotHanlder) sendTimezoneMenu(session *discordgo.Session, intr *discordgo.InteractionCreate, standupID uint) {
	userID := extractUserID(intr)

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

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Welcome to **DailyBot**! I don't know your timezone yet. Please pick one:",
			Flags:   discordgo.MessageFlagsEphemeral,
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
		},
	})
}

func (h *BotHanlder) handleTimezoneSelection(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	var userID string
	if intr.Member != nil {
		userID = intr.Member.User.ID
	} else if intr.User != nil {
		userID = intr.User.ID
	} else if intr.Message != nil && intr.Message.Author != nil {
		userID = intr.Message.Author.ID
	}

	if userID == "" {
		log.Println("Could not determine UserID for timezone selection")
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Something went wrong identifying your user account.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	data := intr.MessageComponentData()
	if len(data.Values) == 0 {
		return
	}
	selectedTZ := data.Values[0]

	var profile models.UserProfile
	h.DB.Where(models.UserProfile{UserID: userID}).FirstOrCreate(&profile)
	profile.Timezone = selectedTZ
	h.DB.Save(&profile)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("✅ Timezone set to `%s`!", selectedTZ),
			Components: []discordgo.MessageComponent{},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	state, err := store.GetState(h.Redis, userID)
	if err == nil && state.StandupID != 0 {
		h.InitiateStandup(session, userID, state.GuildID, intr.ChannelID, state.StandupID)
	} else {
		log.Println("No pending standup state found after timezone selection.")
	}
}
