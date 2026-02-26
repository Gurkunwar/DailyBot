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
	case discordgo.InteractionMessageComponent:
		customID := intr.MessageComponentData().CustomID

		if strings.HasPrefix(customID, "add_next_q_") {
			var nextQ int
			fmt.Sscanf(customID, "add_next_q_%d", &nextQ)
			h.openSingleQuestionModal(session, intr, nextQ)
		} else if customID == "finalize_create_standup" {
			h.finalizeCreateStandup(session, intr)
		} else if strings.HasPrefix(customID, "open_standup_modal_") {
			idStr := strings.TrimPrefix(customID, "open_standup_modal_")
			h.openSingleAnswerModal(session, intr, idStr, 0)
		} else if strings.HasPrefix(customID, "continue_standup_") {
			var standupID uint
			var qIndex int
			fmt.Sscanf(customID, "continue_standup_%d_%d", &standupID, &qIndex)
			h.openSingleAnswerModal(session, intr, fmt.Sprintf("%d", standupID), qIndex)
		} else if customID == "select_tz" {
			h.handleTimezoneSelection(session, intr)
		} else if customID == "select_standup_join" {
			h.handleStandupSelection(session, intr)
		}
	case discordgo.InteractionModalSubmit:
		customID := intr.ModalSubmitData().CustomID
		if strings.HasPrefix(customID, "create_q_modal_") {
			h.handleCreateQuestionSubmit(session, intr, customID)
		} else if strings.HasPrefix(customID, "standup_answer_modal_") {
			var standupID uint
			var qIndex int
			fmt.Sscanf(customID, "standup_answer_modal_%d_%d", &standupID, &qIndex)
			h.handleSingleAnswerSubmit(session, intr, standupID, qIndex)
		}
	case discordgo.InteractionApplicationCommandAutocomplete:
		h.handleAutocomplete(session, intr)
	}
}

func (h *BotHanlder) openSingleAnswerModal(
	session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	standupIDStr string,
	qIndex int) {

	var standupID uint
	fmt.Sscanf(standupIDStr, "%d", &standupID)

	var standup models.Standup
	if err := h.DB.First(&standup, standupID).Error; err != nil {
		log.Println("Error fetching standup for modal:", err)
		return
	}

	if qIndex == 0 {
		state := models.StandupState{
			UserID:    intr.User.ID,
			GuildID:   standup.GuildID,
			StandupID: standup.ID,
			Answers:   []string{},
		}
		store.SaveState(h.Redis, intr.User.ID, state)
	}

	questionText := standup.Questions[qIndex]

	label := questionText
	if len(label) > 45 {
		label = label[:42] + "..."
	}

	placeholder := questionText
	if len(placeholder) > 100 {
		placeholder = placeholder[:97] + "..."
	}

	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("standup_answer_modal_%d_%d", standup.ID, qIndex),
			Title:    fmt.Sprintf("%s (%d/%d)", standup.Name, qIndex+1, len(standup.Questions)),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "answer_text",
							Label:       label,
							Style:       discordgo.TextInputParagraph,
							Required:    true,
							Placeholder: placeholder,
						},
					},
				},
			},
		},
	})

	if err != nil {
		log.Println("Error opening answer modal:", err)
	}
}

func (h *BotHanlder) handleSingleAnswerSubmit(session *discordgo.Session,
	intr *discordgo.InteractionCreate,
	standupID uint,
	qIndex int) {

	answer := intr.ModalSubmitData().Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	state, err := store.GetState(h.Redis, intr.User.ID)
	if err != nil {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Session expired. Please run `/start` to try again.", 
			Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	state.Answers = append(state.Answers, strings.TrimSpace(answer))
	store.SaveState(h.Redis, intr.User.ID, *state)

	var standup models.Standup
	h.DB.First(&standup, standupID)

	nextQIndex := qIndex + 1

	if nextQIndex < len(standup.Questions) {
		session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ **Question %d answered!**\n\nReady for question %d?", qIndex+1, nextQIndex+1),
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    fmt.Sprintf("Next: Question %d", nextQIndex+1),
								Style:    discordgo.PrimaryButton,
								CustomID: fmt.Sprintf("continue_standup_%d_%d", standup.ID, nextQIndex),
							},
						},
					},
				},
			},
		})
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "✅ **Standup complete!** Your team has been notified.",
			Components: []discordgo.MessageComponent{},
		},
	})

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
