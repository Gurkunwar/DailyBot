package bot

import (
	"fmt"
	"log"

	"github.com/Gurkunwar/asyncflow/internal/bot/utils"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/Gurkunwar/asyncflow/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) handleHelp(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	helpText := "💡 **AsyncFlow Help Menu**\n\n" +
		"**👤 User Commands**\n" +
		"`/start` - Manually trigger your daily standup form.\n" +
		"`/history` - View past standup reports.\n" +
		"`/timezone` - Set your local timezone so reminders trigger at your morning.\n" +
		"`/poll` - 📊 Create a native poll for your team instantly.\n" +
		"`/delete-my-data` - Permanently delete your profile and leave all standups.\n" +
		"> *💡 Tip: When you receive your automated DM, you can use the **Skip Today** button if you are out of office!*\n\n" +
		"**🛠️ Manager Commands (Admin Only)**\n" +
		"`/create-standup` - Create a new team standup.\n" +
		"`/edit-standup` - Edit Questions, Active Days, Trigger Time, and Report Channel.\n" +
		"`/standup-info` - View all settings, members, and questions for a standup.\n" +
		"`/add-member` - Add a user to an existing standup.\n" +
		"`/remove-member` - Remove a user from an existing standup.\n" +
		"`/delete-standup` - Permanently delete an existing standup team.\n\n" +
		"**📋 Poll Management (Admin Only)**\n" +
		"`/poll-list` - List all recent polls and get their IDs.\n" +
		"`/poll-audit` - See a detailed breakdown of who voted for what.\n" +
		"`/poll-export` - Download poll results to a CSV/Excel file.\n" +
		"`/poll-end` - Manually lock a live poll early.\n\n" +
		"ℹ️ *Note: I will automatically ping your team members at their local time on your selected active days!*"

	utils.RespondWithMessage(session, intr, helpText, true)
}

func (h *BotHanlder) handleDeleteMyData(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := utils.ExtractUserID(intr)

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

func (h *BotHanlder) sendTimezoneMenu(session *discordgo.Session, intr *discordgo.InteractionCreate, standupID uint) {
    userID := utils.ExtractUserID(intr)

    state := models.StandupState{
        UserID:    userID,
        StandupID: standupID,
    }
    store.SaveState(h.Redis, userID, state)

    options := []discordgo.SelectMenuOption{
        {Label: "Universal Time (UTC)", Value: "UTC", Description: "Standard Global Time"},
        {Label: "US Pacific (PST/PDT)", Value: "America/Los_Angeles", Description: "UTC-8:00 / UTC-7:00"},
        {Label: "US Central (CST/CDT)", Value: "America/Chicago", Description: "UTC-6:00 / UTC-5:00"},
        {Label: "US East (EST/EDT)", Value: "America/New_York", Description: "UTC-5:00 / UTC-4:00"},
        {Label: "London (GMT/BST)", Value: "Europe/London", Description: "UTC+0:00 / UTC+1:00"},
        {Label: "Europe Central (CET)", Value: "Europe/Paris", Description: "UTC+1:00 / UTC+2:00"},
        {Label: "India (IST)", Value: "Asia/Kolkata", Description: "UTC+5:30"},
        {Label: "Singapore (SGT)", Value: "Asia/Singapore", Description: "UTC+8:00"},
        {Label: "Japan (JST)", Value: "Asia/Tokyo", Description: "UTC+9:00"},
        {Label: "Australia East (AEST)", Value: "Australia/Sydney", Description: "UTC+10:00 / UTC+11:00"},
    }

    session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "Welcome to **AsyncFlow**! I don't know your timezone yet. Please pick one:",
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
	userID := utils.ExtractUserID(intr)

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

	utils.UpdateMessage(session, intr, fmt.Sprintf("✅ Timezone set to `%s`!", selectedTZ), nil)

	state, err := store.GetState(h.Redis, userID)
	if err == nil && state.StandupID != 0 {
		h.Standups.InitiateStandup(session, userID, state.GuildID, intr.ChannelID, state.StandupID)
	} else {
		log.Println("No pending standup state found after timezone selection.")
	}
}
