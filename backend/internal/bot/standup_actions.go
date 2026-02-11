package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) InitiateStandup(s *discordgo.Session, userID, guildID string) {
	var config models.Guild
	h.DB.Where("guild_id = ?", guildID).First(&config)
	if config.ReportChannelID == "" {
		s.ChannelMessageSend(guildID, "⚠️ Set a report channel first!")
		return
	}

	var profile models.UserProfile
	h.DB.Where("user_id = ?", userID).First(&profile)

	if profile.UserID == "" {
        profile.UserID = userID
        profile.PrimaryGuildID = guildID
        h.DB.Create(&profile)
    } else if profile.PrimaryGuildID == "" {
        profile.PrimaryGuildID = guildID
        h.DB.Save(&profile)
    }
	
	channel, _ := s.UserChannelCreate(userID)

	if profile.Timezone == "" || profile.Timezone == "UTC" {
		h.sendTimezoneMenu(s, channel.ID, userID, guildID)
		return
	}

	h.startQuestionFlow(s, channel.ID, userID, guildID)
}

func (h *BotHanlder) finalizeStandup(s *discordgo.Session, state *models.StandupState) {
	var config models.Guild
	result := h.DB.Where("guild_id = ?", state.GuildID).First(&config)

	if result.Error != nil || config.ReportChannelID == "" {
		log.Printf("Could not find report channel for guild %s", state.GuildID)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🚀 Daily Standup Update",
		Description: fmt.Sprintf("Progress report from <@%s>", state.UserID),
		Color:       0x5865F2,
		// Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Yesterday", Value: "✅ " + state.Answers[0], Inline: false},
			{Name: "Today", Value: "📅 " + state.Answers[1], Inline: false},
			{Name: "Blockers", Value: "🚫 " + state.Answers[2], Inline: false},
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(config.ReportChannelID, embed)
}

func (h *BotHanlder) sendTimezoneMenu(s *discordgo.Session, channelID, userID, guildID string) {
	state := models.StandupState{
		UserID:      userID,
		GuildID:     guildID,
		CurrentStep: -1,
	}
	store.SaveState(h.Redis, userID, state)

	options := []discordgo.SelectMenuOption{
		{Label: "India (IST)", Value: "Asia/Kolkata", Description: "UTC+5:30"},
		{Label: "US East (EST)", Value: "America/New_York", Description: "UTC-5:00"},
		{Label: "London (GMT)", Value: "Europe/London", Description: "UTC+0:00"},
		{Label: "Singapore (SGT)", Value: "Asia/Singapore", Description: "UTC+8:00"},
	}

	s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
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

func (h *BotHanlder) startQuestionFlow(session *discordgo.Session, channelID, userID, guildID string) {
	state := models.StandupState{
		UserID:      userID,
		GuildID:     guildID,
		CurrentStep: 0,
		Answers:     []string{},
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
						CustomID: "open_standup_modal",
					},
				},
			},
		},
	})
}
