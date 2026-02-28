package bot

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (h *BotHanlder) OnInteraction(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		userID := extractUserID(intr)
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
		case "delete-my-data":
			h.handleDeleteMyData(session, intr)
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
			h.sendTimezoneMenu(session, intr, 0)
		}

	case discordgo.InteractionMessageComponent:
		customID := intr.MessageComponentData().CustomID

		if strings.HasPrefix(customID, "add_next_q_") {
			var nextQ int
			fmt.Sscanf(customID, "add_next_q_%d", &nextQ)
			h.openSingleQuestionModal(session, intr, nextQ)

		} else if strings.HasPrefix(customID, "skip_standup_") {
			var standupID uint
			fmt.Sscanf(customID, "skip_standup_%d", &standupID)
			h.handleSkipStandup(session, intr, standupID)
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

		} else if strings.HasPrefix(customID, "open_q_dash_") {
			var standupID uint
			fmt.Sscanf(customID, "open_q_dash_%d", &standupID)
			h.showQuestionDashboard(session, intr, standupID, true)

		} else if strings.HasPrefix(customID, "select_q_") {
			var standupID uint
			fmt.Sscanf(customID, "select_q_%d", &standupID)
			var qIndex int
			fmt.Sscanf(intr.MessageComponentData().Values[0], "%d", &qIndex)
			h.handleEditSingleQuestionPrompt(session, intr, standupID, qIndex)

		} else if strings.HasPrefix(customID, "add_q_btn_") {
			var standupID uint
			fmt.Sscanf(customID, "add_q_btn_%d", &standupID)
			h.handleAddQuestionPrompt(session, intr, standupID)
		} else if strings.HasPrefix(customID, "finish_q_dash_") {
			var standupID uint
			fmt.Sscanf(customID, "finish_q_dash_%d", &standupID)
			h.handleFinishQuestionEdit(session, intr, standupID)
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

		} else if strings.HasPrefix(customID, "edit_single_q_") {
			var standupID uint
			var qIndex int
			fmt.Sscanf(customID, "edit_single_q_%d_%d", &standupID, &qIndex)
			h.handleQuestionSubmit(session, intr, standupID, qIndex, false)

		} else if strings.HasPrefix(customID, "add_single_q_") {
			var standupID uint
			fmt.Sscanf(customID, "add_single_q_%d", &standupID)
			h.handleQuestionSubmit(session, intr, standupID, 0, true)
		}

	case discordgo.InteractionApplicationCommandAutocomplete:
		h.handleAutocomplete(session, intr)
	}
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

		userID := extractUserID(intr)

		var standups []models.Standup

		if isServerAdmin(intr) {
			h.DB.Where("guild_id = ?", intr.GuildID).Find(&standups)
		} else {
			h.DB.Where("guild_id = ? AND manager_id = ?", intr.GuildID, userID).Find(&standups)
		}

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
