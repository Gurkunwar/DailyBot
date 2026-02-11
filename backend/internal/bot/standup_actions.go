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
    // 1. Find or Create the User
    h.DB.Unscoped().Preload("Standups").Where("user_id = ?", userID).First(&profile)

    if profile.ID == 0 {
        profile.UserID = userID
        h.DB.Create(&profile)
    } else if profile.DeletedAt.Valid {
        h.DB.Model(&profile).Unscoped().Update("deleted_at", nil)
    }

    var targetStandup models.Standup

    // Case A: Automated Worker OR Selected from Menu (Specific ID provided)
    if standupID != 0 {
        if err := h.DB.First(&targetStandup, standupID).Error; err != nil {
            return 
        }
    } else {
        // Case B: Manual /start command
        // Instead of checking what they joined, check what is AVAILABLE in the guild
        var guildStandups []models.Standup
        h.DB.Where("guild_id = ?", guildID).Find(&guildStandups)

        if len(guildStandups) == 0 {
            s.ChannelMessageSend(userID, "⚠️ No standups found in this server. Admin needs to run `/create-standup`.")
            return
        } else if len(guildStandups) == 1 {
            // Only 1 option? Auto-select it.
            targetStandup = guildStandups[0]
            
            // Auto-join them if they aren't linked yet
            joined := false
            for _, s := range profile.Standups {
                if s.ID == targetStandup.ID {
                    joined = true
                    break
                }
            }
            if !joined {
                 h.DB.Model(&profile).Association("Standups").Append(&targetStandup)
            }
        } else {
            // More than 1 option exists? ALWAYS show the menu.
            h.sendStandupSelectionMenu(s, userID, guildID, guildStandups)
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

	// Split questions by semicolon
	questions := strings.Split(questionsRaw, ";")

	standup := models.Standup{
		Name:            name,
		GuildID:         i.GuildID,
		ManagerID:       i.Member.User.ID,
		ReportChannelID: channelID,
		Questions:       questions,
	}

	if err := h.DB.Create(&standup).Error; err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Error creating standup."},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Standup **%s** created! Users can now join using ID: `%d`", name, standup.ID),
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