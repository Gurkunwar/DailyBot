package poll

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/Gurkunwar/dailybot/internal/store"
	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) handleInitPoll(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := intr.Member.User.ID

	store.ClearPollDraft(h.Redis, userID)
	freshState := models.PollState{}
	store.SavePollDraft(h.Redis, userID, freshState)

	embed, components := h.renderDashboard(freshState)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *PollHandler) renderDashboard(state models.PollState) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	qText := "📝 *No question set yet. Click 'Set Question' below.*"
	if state.Question != "" {
		qText = fmt.Sprintf("**%s**", state.Question)
	}

	var optBuilder strings.Builder
	if len(state.Options) == 0 {
		optBuilder.WriteString("*No options added yet.*")
	} else {
		for i, opt := range state.Options {
			optBuilder.WriteString(fmt.Sprintf("**%d.** %s\n", i+1, opt))
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: "🛠️ Interactive Poll Builder",
		Color: 0xFEE75C,
		Description: fmt.Sprintf("Use the buttons below to build your poll!\n\n**Question:**\n%s\n\n**Options:**\n%s",
			qText, optBuilder.String()),
	}

	canPublish := state.Question != "" && len(state.Options) >= 2
	canAddOption := len(state.Options) < 10

	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "📝 Set Question",
			Style:    discordgo.SecondaryButton,
			CustomID: "poll_btn_question",
		},
		discordgo.Button{
			Label:    "➕ Add Option",
			Style:    discordgo.PrimaryButton,
			CustomID: "poll_btn_option",
			Disabled: !canAddOption,
		},
		discordgo.Button{
			Label:    "❌ Cancel",
			Style:    discordgo.DangerButton,
			CustomID: "poll_btn_cancel",
		},
		discordgo.Button{
			Label:    "🚀 Publish",
			Style:    discordgo.SuccessButton,
			CustomID: "poll_btn_publish",
			Disabled: !canPublish,
		},
	}

	return embed, []discordgo.MessageComponent{discordgo.ActionsRow{Components: buttons}}
}

func (h *PollHandler) promptQuestionModal(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "poll_modal_question",
			Title:    "Set Poll Question",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "question_text",
							Label:       "What are we voting on?",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g., What should we order for lunch?",
							Required:    true,
							MaxLength:   250,
						},
					},
				},
			},
		},
	})
}

func (h *PollHandler) promptOptionModal(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "poll_modal_option",
			Title:    "Add Poll Option",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "option_text",
							Label:       "Option text",
							Style:       discordgo.TextInputShort,
							Placeholder: "e.g., Pepperoni Pizza 🍕",
							Required:    true,
							MaxLength:   80,
						},
					},
				},
			},
		},
	})
}

func (h *PollHandler) saveQuestionFromModal(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := intr.Member.User.ID

	questionText := intr.ModalSubmitData().
		Components[0].(*discordgo.ActionsRow).
		Components[0].(*discordgo.TextInput).Value

	state, _ := store.GetPollDraft(h.Redis, userID)

	state.Question = strings.TrimSpace(questionText)
	store.SavePollDraft(h.Redis, userID, *state)

	embed, components := h.renderDashboard(*state)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

func (h *PollHandler) saveOptionFromModal(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := intr.Member.User.ID
	optionText := intr.ModalSubmitData().
		Components[0].(*discordgo.ActionsRow).
		Components[0].(*discordgo.TextInput).Value

	state, _ := store.GetPollDraft(h.Redis, userID)

	if len(state.Options) < 10 {
		state.Options = append(state.Options, strings.TrimSpace(optionText))
		store.SavePollDraft(h.Redis, userID, *state)
	}

	embed, components := h.renderDashboard(*state)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}