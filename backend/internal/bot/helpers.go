package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func extractUserID(intr *discordgo.InteractionCreate) string {
	if intr.Member != nil {
		return intr.Member.User.ID
	}
	if intr.User != nil {
		return intr.User.ID
	}
	if intr.Message != nil && intr.Message.Author != nil {
		return intr.Message.Author.ID
	}
	return ""
}

func respondWithError(session *discordgo.Session, interaction *discordgo.Interaction, message string) {
	session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func isServerAdmin(intr *discordgo.InteractionCreate) bool {
	if intr.Member == nil {
		return false
	}

	return intr.Member.Permissions&discordgo.PermissionAdministrator != 0
}

func formatLocalTime(dbTimeStr string, userTZ string) string {
	if userTZ == "" {
		return fmt.Sprintf("**%s (Your Local Time)**\n> ⚠️ *Wait! You haven't set a timezone yet. Run `/timezone` so this triggers at your actual morning!*", dbTimeStr)
	}

	return fmt.Sprintf("**%s** (%s)", dbTimeStr, userTZ)
}