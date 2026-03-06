package poll

import (
	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) PollRouter(session *discordgo.Session, intr *discordgo.InteractionCreate) bool {
	if intr.Type == discordgo.InteractionApplicationCommand {
		cmdName := intr.ApplicationCommandData().Name

		switch cmdName {
		case "poll":
			h.handleCreateNativePoll(session, intr)
			return true
		case "poll-audit":
			h.HandlePollAudit(session, intr)
			return true
		case "poll-end":
			h.HandlePollEnd(session, intr)
			return true
		case "poll-export":
			h.HandlePollExport(session, intr)
			return true
		case "poll-list":
			h.handlePollList(session, intr)
		case "poll-delete":
			h.HandlePollDelete(session, intr)
			return true
		}
	}

	return false
}
