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

	state.CurrentStep = 0
	store.SaveState(h.Redis, userID, *state)

	h.startQuestionFlow(session, intr.ChannelID, userID, state.GuildID)
}

func (h *BotHanlder) OnInteraction(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		data := intr.ApplicationCommandData()
		switch data.Name {
		case "start":
			h.InitiateStandup(session, intr.Member.User.ID, intr.GuildID)
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
		}
	case discordgo.InteractionMessageComponent:
		if intr.MessageComponentData().CustomID == "open_standup_modal" {
			h.openStandupModal(session, intr)
		}
		if intr.MessageComponentData().CustomID == "select_tz" {
			h.handleTimezoneSelection(session, intr)
		}

	case discordgo.InteractionModalSubmit:
		h.handleModalSubmit(session, intr)
	}
}

func (h *BotHanlder) openStandupModal(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	err := session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "standup_modal",
			Title:    "Daily Standup Form",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "yesterday", Label: "What did you do yesterday?",
						Style: discordgo.TextInputParagraph, Required: true,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "today", Label: "What are you planning to do today?",
						Style: discordgo.TextInputParagraph, Required: true,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID: "blockers", Label: "Any blockers in your way?",
						Style: discordgo.TextInputParagraph, Value: "None",
					},
				}},
			},
		},
	})

	if err != nil {
		log.Println("Error opening modal:", err)
	}
}

func (h *BotHanlder) handleModalSubmit(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	data := intr.ModalSubmitData()

	yesterday := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	today := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	blockers := data.Components[2].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value

	state, err := store.GetState(h.Redis, intr.User.ID)
	if err != nil {
		log.Println("Error retrieving state from Redis:", err)
		return
	}
	state.Answers = []string{yesterday, today, blockers}

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
