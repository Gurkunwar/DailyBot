package poll

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) PollRouter(session *discordgo.Session, intr *discordgo.InteractionCreate) bool {
	switch intr.Type {
	case discordgo.InteractionApplicationCommand:
		if intr.ApplicationCommandData().Name == "poll" {
			h.handleInitPoll(session, intr)
			return true
		}
	case discordgo.InteractionMessageComponent:
		customID := intr.MessageComponentData().CustomID

		if strings.HasPrefix(customID, "vote_") {
			h.handleVote(session, intr)
			return true
		} else if strings.HasPrefix(customID, "poll_btn_end_") {
			h.handleEndPoll(session, intr)
			return true
		}

		switch customID {
		case "poll_btn_question":
			h.promptQuestionModal(session, intr)
			return true
		case "poll_btn_option":
			h.promptOptionModal(session, intr)
			return true
		case "poll_btn_cancel":
			h.handleCancelPoll(session, intr)
			return true
		case "poll_btn_publish":
			h.handlePublishPoll(session, intr)
			return true
		}
		return false
	case discordgo.InteractionModalSubmit:
		customID := intr.ModalSubmitData().CustomID

		switch customID {
		case "poll_modal_question":
			h.saveQuestionFromModal(session, intr)
			return true
		case "poll_modal_option":
			h.saveOptionFromModal(session, intr)
			return true
		}
	}

	return false
}
