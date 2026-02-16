package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) InitiateStandup(s *discordgo.Session, userID string, guildID string, standupID uint) {
	var profile models.UserProfile
	// 1. Fetch User and their PRE-APPROVED Standups
	h.DB.Unscoped().Preload("Standups").Where("user_id = ?", userID).First(&profile)

	// Revive if deleted
	if profile.ID != 0 && profile.DeletedAt.Valid {
		h.DB.Model(&profile).Unscoped().Update("deleted_at", nil)
	}

	if profile.ID == 0 || len(profile.Standups) == 0 {
		s.ChannelMessageSend(userID,
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
			h.sendStandupSelectionMenu(s, userID, guildID, profile.Standups)
			return
		}
	}

	if targetStandup.ReportChannelID == "" {
		s.ChannelMessageSend(userID, "⚠️ This standup has no report channel set.")
		return
	}

	channel, _ := s.UserChannelCreate(userID)
	if profile.Timezone == "" || profile.Timezone == "UTC" {
		h.sendTimezoneMenu(s, channel.ID, userID, targetStandup.ID)
		return
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

func (h *BotHanlder) handleCreateStandup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	name := options[0].StringValue()
	channelID := options[1].ChannelValue(s).ID
	questionsRaw := options[2].StringValue()
	membersRaw := options[3].StringValue()
	standupTime := "9:00"

	if len(options) > 4 {
		standupTime = options[4].StringValue()
	}

	questions := strings.Split(questionsRaw, ";")

	standup := models.Standup{
		Name:            name,
		GuildID:         i.GuildID,
		ManagerID:       i.Member.User.ID,
		ReportChannelID: channelID,
		Questions:       questions,
		Time:            standupTime,
	}

	if err := h.DB.Create(&standup).Error; err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Error creating standup."},
		})
		return
	}

	members := strings.Fields(membersRaw)
	addedCount := 0

	for _, member := range members {
		if strings.HasPrefix(member, "<@") && strings.HasSuffix(member, ">") {
			userID := strings.Trim(member, "<@!>")

			var user models.UserProfile
			h.DB.FirstOrCreate(&user, models.UserProfile{UserID: userID})
			h.DB.Model(&user).Association("Standups").Append(&standup)
			addedCount++

			dmChannel, err := s.UserChannelCreate(userID)
			if err == nil {
				welcomeMsg := fmt.Sprintf(
					"👋 **You've been added to the '%s' Standup!**\n\n"+
						"You can now submit your daily reports for this team.\n"+
						"Run `/start` here or in the server to begin.",
					name,
				)
				s.ChannelMessageSend(dmChannel.ID, welcomeMsg)
			} else {
				fmt.Printf("Could not DM user %s: %v\n", userID, err)
			}
		}
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Standup **%s** created with **%d** members!", name, addedCount),
		},
	})
}

func (h *BotHanlder) sendTimezoneMenu(s *discordgo.Session, channelID, userID string, standupID uint) {
	state := models.StandupState{
		UserID:      userID,
		StandupID:   standupID,
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

func (h *BotHanlder) startQuestionFlow(session *discordgo.Session, channelID, userID string, standup models.Standup) {
	state := models.StandupState{
		UserID:      userID,
		GuildID:     standup.GuildID,
		StandupID:   standup.ID,
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

func (h *BotHanlder) sendStandupSelectionMenu(s *discordgo.Session, userID, guildID string, standups []models.Standup) {
	channel, _ := s.UserChannelCreate(userID)

	var options []discordgo.SelectMenuOption
	for _, st := range standups {
		options = append(options, discordgo.SelectMenuOption{
			Label:       st.Name,
			Value:       fmt.Sprintf("%d", st.ID),
			Description: fmt.Sprintf("ID: %d", st.ID),
		})
	}

	s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Content: "found multiple standups in this server. Please select one:",
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
	// 1. Parse the selected Standup ID
	selectedID := intr.MessageComponentData().Values[0] // e.g., "1"

	// 2. Link the user to this standup
	var standup models.Standup
	h.DB.First(&standup, selectedID)

	var user models.UserProfile
	h.DB.Preload("Standups").Where("user_id = ?", intr.User.ID).First(&user)

	// Add to participants list
	h.DB.Model(&user).Association("Standups").Append(&standup)

	// 3. Acknowledge and Start
	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("✅ You joined **%s**!", standup.Name),
			Components: []discordgo.MessageComponent{}, // Remove the dropdown
		},
	})

	h.InitiateStandup(session, intr.User.ID, standup.GuildID, standup.ID)
}
