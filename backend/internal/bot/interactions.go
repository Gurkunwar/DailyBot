package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) handleTimezoneSelection(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	selectedTZ := intr.MessageComponentData().Values[0]
	userID := intr.User.ID
	if userID == "" && intr.Member != nil {
		userID = intr.Member.User.ID
	}

	var profile models.UserProfile
	h.DB.Where(models.UserProfile{UserID: userID}).FirstOrCreate(&profile)
	profile.Timezone = selectedTZ
	h.DB.Save(&profile)

	state, err := store.GetState(h.Redis, userID)
	if err != nil {
		log.Println("Error fetching state after TZ selection:", err)
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("✅ Timezone set to `%s`!", selectedTZ),
			Components: []discordgo.MessageComponent{},
		},
	})

	var standup models.Standup
	h.DB.First(&standup, state.StandupID)

	state.CurrentStep = 0
	store.SaveState(h.Redis, userID, *state)

	h.startQuestionFlow(session, intr.ChannelID, userID, standup)
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
			h.InitiateStandup(session, userID, intr.GuildID, intr.ChannelID, 0)
			session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Standup started!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		case "help":
			h.handleHelp(session, intr)
		case "reset":
			h.handleReset(session, intr)
		case "set-channel":
			h.handleSetChannel(session, intr)
		case "create-standup":
			h.handleCreateStandup(session, intr)
		}
	case discordgo.InteractionMessageComponent:
		switch intr.MessageComponentData().CustomID {
		case "open_standup_modal":
			h.openStandupModal(session, intr)
		case "select_tz":
			h.handleTimezoneSelection(session, intr)
		case "select_standup_join":
			h.handleStandupSelection(session, intr)
		}
	case discordgo.InteractionModalSubmit:
		h.handleModalSubmit(session, intr)
	}
}

func (h *BotHanlder) openStandupModal(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	state, err := store.GetState(h.Redis, intr.User.ID)
	if err != nil {
		return
	}

	var standup models.Standup
	if err := h.DB.First(&standup, state.StandupID).Error; err != nil {
		log.Println("Error fetching standup for modal:", err)
		return
	}

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

	err = session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
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
