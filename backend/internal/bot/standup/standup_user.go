package standup

import (
	"fmt"
	// "log"
	"time"

	"github.com/Gurkunwar/dailybot/internal/bot/utils"
	"github.com/Gurkunwar/dailybot/internal/models"
	// "github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

// func (h *StandupHandler) handleHelp(session *discordgo.Session, intr *discordgo.InteractionCreate) {
// 	helpText := "💡 **DailyBot Help Menu**\n\n" +
// 		"**👤 User Commands**\n" +
// 		"`/start` - Manually trigger your daily standup form.\n" +
// 		"`/history` - View your past standup reports.\n" +
// 		"`/timezone` - Set your local timezone so reminders trigger at your morning.\n" +
// 		"`/delete-my-data` - Permanently delete your profile and leave all standups.\n" +
// 		"> *💡 Tip: When you receive your automated DM, you can use the **Skip Today** button if you are out of office!*\n\n" +
// 		"**🛠️ Manager Commands (Admin Only)**\n" +
// 		"`/create-standup` - Create a new team standup.\n" +
// 		"`/standup-info` - 📊 View all settings, members, and questions for a standup.\n" +
// 		"`/edit-standup` - **[Dashboard]** Edit Questions, Active Days, Trigger Time, and Report Channel.\n" +
// 		"`/add-member` - Add a user to an existing standup.\n" +
// 		"`/remove-member` - Remove a user from an existing standup.\n" +
// 		"`/delete-standup` - Permanently delete an existing standup team.\n\n" +
// 		"ℹ️ *Note: I will automatically ping your team members at their local time on your selected active days!*"

// 	utils.RespondWithMessage(session, intr, helpText, true)
// }

// func (h *StandupHandler) handleDeleteMyData(session *discordgo.Session, intr *discordgo.InteractionCreate) {
// 	userID := utils.ExtractUserID(intr)

// 	var user models.UserProfile
// 	if err := h.DB.Unscoped().Where("user_id = ?", userID).First(&user).Error; err != nil {
// 		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
// 			Type: discordgo.InteractionResponseChannelMessageWithSource,
// 			Data: &discordgo.InteractionResponseData{
// 				Content: "❌ No profile found to reset.",
// 				Flags:   discordgo.MessageFlagsEphemeral,
// 			},
// 		})
// 		return
// 	}

// 	if err := h.DB.Model(&user).Association("Standups").Clear(); err != nil {
// 		log.Println("Error clearing standup teams:", err)
// 	}

// 	if result := h.DB.Unscoped().Delete(&user); result.Error != nil {
// 		session.ChannelMessageSend(intr.ChannelID, "❌ Failed to reset profile.")
// 		return
// 	}

// 	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
// 		Data: &discordgo.InteractionResponseData{
// 			Content: "✅ **Profile Reset Complete.** You have been removed from all standup teams.",
// 			Flags:   discordgo.MessageFlagsEphemeral,
// 		},
// 	})
// }

func (h *StandupHandler) handleHistory(session *discordgo.Session, intr *discordgo.InteractionCreate) {
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

	callerID := utils.ExtractUserID(intr)

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

// func (h *StandupHandler) sendTimezoneMenu(session *discordgo.Session, intr *discordgo.InteractionCreate, standupID uint) {
// 	userID := utils.ExtractUserID(intr)

// 	state := models.StandupState{
// 		UserID:    userID,
// 		StandupID: standupID,
// 	}
// 	store.SaveState(h.Redis, userID, state)

// 	options := []discordgo.SelectMenuOption{
// 		{Label: "India (IST)", Value: "Asia/Kolkata", Description: "UTC+5:30"},
// 		{Label: "US East (EST)", Value: "America/New_York", Description: "UTC-5:00"},
// 		{Label: "London (GMT)", Value: "Europe/London", Description: "UTC+0:00"},
// 		{Label: "Singapore (SGT)", Value: "Asia/Singapore", Description: "UTC+8:00"},
// 	}

// 	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
// 		Data: &discordgo.InteractionResponseData{
// 			Content: "Welcome to **DailyBot**! I don't know your timezone yet. Please pick one:",
// 			Flags:   discordgo.MessageFlagsEphemeral,
// 			Components: []discordgo.MessageComponent{
// 				discordgo.ActionsRow{
// 					Components: []discordgo.MessageComponent{
// 						discordgo.SelectMenu{
// 							CustomID:    "select_tz",
// 							Placeholder: "Select your local timezone",
// 							Options:     options,
// 						},
// 					},
// 				},
// 			},
// 		},
// 	})
// }

// func (h *StandupHandler) handleTimezoneSelection(session *discordgo.Session, intr *discordgo.InteractionCreate) {
// 	userID := utils.ExtractUserID(intr)

// 	if userID == "" {
// 		log.Println("Could not determine UserID for timezone selection")
// 		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
// 			Type: discordgo.InteractionResponseChannelMessageWithSource,
// 			Data: &discordgo.InteractionResponseData{
// 				Content: "❌ Something went wrong identifying your user account.",
// 				Flags:   discordgo.MessageFlagsEphemeral,
// 			},
// 		})
// 		return
// 	}

// 	data := intr.MessageComponentData()
// 	if len(data.Values) == 0 {
// 		return
// 	}
// 	selectedTZ := data.Values[0]

// 	var profile models.UserProfile
// 	h.DB.Where(models.UserProfile{UserID: userID}).FirstOrCreate(&profile)
// 	profile.Timezone = selectedTZ
// 	h.DB.Save(&profile)

// 	utils.UpdateMessage(session, intr, fmt.Sprintf("✅ Timezone set to `%s`!", selectedTZ), nil)

// 	state, err := store.GetState(h.Redis, userID)
// 	if err == nil && state.StandupID != 0 {
// 		h.InitiateStandup(session, userID, state.GuildID, intr.ChannelID, state.StandupID)
// 	} else {
// 		log.Println("No pending standup state found after timezone selection.")
// 	}
// }
