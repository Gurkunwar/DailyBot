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
		},
	})

	state, err := store.GetState(h.Redis, userID)
	if err == nil && state.StandupID != 0 {
		h.InitiateStandup(session, userID, state.GuildID, intr.ChannelID, state.StandupID)
	} else {
		log.Println("No pending standup state found after timezone selection.")
	}
}

func (h *BotHanlder) OnInteraction(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		var userID string
		if intr.Member != nil {
			userID = intr.Member.User.ID
		} else {
			userID = intr.User.ID
		}
		data := intr.ApplicationCommandData()

		switch data.Name {
		case "start":
			session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Standup started!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			h.InitiateStandup(session, userID, intr.GuildID, intr.ChannelID, 0)
		case "help":
			h.handleHelp(session, intr)
		case "reset":
			h.handleReset(session, intr)
		case "set-channel":
			h.handleSetChannel(session, intr)
		case "create-standup":
			h.handleCreateStandup(session, intr)
		case "edit-standup":
			h.handleEditStandup(session, intr)
		case "delete-standup":
			h.handleDeleteStandup(session, intr)
		case "add-member":
			h.handleAddMember(session, intr)
		case "remove-member":
			h.handleRemoveMember(session, intr)
		case "history":
			h.handleHistory(session, intr)
		case "timezone":
			h.sendTimezoneMenu(session, intr.ChannelID, userID, 0)
		}
	// case discordgo.InteractionMessageComponent:
	// 	switch intr.MessageComponentData().CustomID {
	// 	case "open_standup_modal":
	// 		h.openStandupModal(session, intr)
	// 	case "select_tz":
	// 		h.handleTimezoneSelection(session, intr)
	// 	case "select_standup_join":
	// 		h.handleStandupSelection(session, intr)
	// 	}
	case discordgo.InteractionMessageComponent:
		customID := intr.MessageComponentData().CustomID

		if strings.HasPrefix(customID, "open_standup_modal_") {
			idStr := strings.TrimPrefix(customID, "open_standup_modal_")
			h.openStandupModal(session, intr, idStr) // Pass it to your modal function
		} else if customID == "select_tz" {
			h.handleTimezoneSelection(session, intr)
		} else if customID == "select_standup_join" {
			h.handleStandupSelection(session, intr)
		}
	case discordgo.InteractionModalSubmit:
		h.handleModalSubmit(session, intr)
	case discordgo.InteractionApplicationCommandAutocomplete:
		h.handleAutocomplete(session, intr)
	}
}

// func (h *BotHanlder) openStandupModal(session *discordgo.Session, intr *discordgo.InteractionCreate) {
// 	state, err := store.GetState(h.Redis, intr.User.ID)
// 	if err != nil {
// 		return
// 	}

// 	var standup models.Standup
// 	if err := h.DB.First(&standup, state.StandupID).Error; err != nil {
// 		log.Println("Error fetching standup for modal:", err)
// 		return
// 	}

// 	var components []discordgo.MessageComponent
// 	for i, question := range standup.Questions {
// 		if i >= 5 {
// 			break
// 		}

// 		components = append(components, discordgo.ActionsRow{
// 			Components: []discordgo.MessageComponent{
// 				discordgo.TextInput{
// 					CustomID:    fmt.Sprintf("answer_%d", i),
// 					Label:       question,
// 					Style:       discordgo.TextInputParagraph,
// 					Required:    true,
// 					Placeholder: "Type your answer here...",
// 				},
// 			},
// 		})
// 	}

// 	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseModal,
// 		Data: &discordgo.InteractionResponseData{
// 			CustomID:   "standup_modal",
// 			Title:      fmt.Sprintf("%s Submission", standup.Name),
// 			Components: components,
// 		},
// 	})

// 	if err != nil {
// 		log.Println("Error opening modal:", err)
// 	}
// }

func (h *BotHanlder) openStandupModal(session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	standupIDStr string) {
		
	var standupID uint
	fmt.Sscanf(standupIDStr, "%d", &standupID)

	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		log.Println("Error fetching standup for modal:", err)
		return
	}

	state := models.StandupState{
		UserID:    intr.User.ID,
		GuildID:   standup.GuildID,
		StandupID: standup.ID,
		Answers:   []string{},
	}
	store.SaveState(h.Redis, intr.User.ID, state)

	var components []discordgo.MessageComponent
	for i, question := range standup.Questions {
		if i >= 5 {
			break
		}

		components = append(components, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.TextInput{
					CustomID:    fmt.Sprintf("answer_%d", i),
					Label:       question,
					Style:       discordgo.TextInputParagraph,
					Required:    true,
					Placeholder: "Type your answer here...",
				},
			},
		})
	}

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID:   "standup_modal",
			Title:      fmt.Sprintf("%s Submission", standup.Name),
			Components: components,
		},
	})

	if err != nil {
		log.Println("Error opening modal:", err)
	}
}

func (h *BotHanlder) handleModalSubmit(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	data := intr.ModalSubmitData()
	var answers []string

	for _, comp := range data.Components {
		if row, ok := comp.(*discordgo.ActionsRow); ok {
			if textInput, ok := row.Components[0].(*discordgo.TextInput); ok {
				answers = append(answers, textInput.Value)
			}
		}
	}

	state, err := store.GetState(h.Redis, intr.User.ID)
	if err != nil {
		log.Println("Error retrieving state from Redis:", err)
		return
	}
	state.Answers = answers

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Standup submitted! Your team has been notified.",
		},
	})

	if state.GuildID == "" {
		log.Println("No Guild ID in state")
		return
	}

	h.finalizeStandup(session, state)
	h.Redis.Del(context.Background(), "state:"+intr.User.ID)
}

func (h *BotHanlder) handleAutocomplete(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	data := intr.ApplicationCommandData()
	choices := []*discordgo.ApplicationCommandOptionChoice{}

	var typedValue string
	for _, opt := range data.Options {
		if opt.Focused {
			typedValue = strings.ToLower(opt.StringValue())
			break
		}
	}

	if data.Name == "delete-standup" ||
		data.Name == "add-member" ||
		data.Name == "remove-member" ||
		data.Name == "set-channel" ||
		data.Name == "edit-standup" ||
		data.Name == "history" {

		var userID string
		if intr.Member != nil {
			userID = intr.Member.User.ID
		} else {
			userID = intr.User.ID
		}

		var standups []models.Standup
		h.DB.Where("guild_id = ? AND manager_id = ?", intr.GuildID, userID).Find(&standups)

		for _, st := range standups {
			if strings.Contains(strings.ToLower(st.Name), typedValue) {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
					Name:  st.Name,
					Value: st.Name,
				})
			}
		}
	}

	if len(choices) > 25 {
		choices = choices[:25]
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})
}
