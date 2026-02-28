package bot

import (
	"fmt"
	"time"

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

func getUserLocalTime(tz string) time.Time {
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		loc = time.UTC
	}
	return time.Now().In(loc)
}

func respondWithMessage(session *discordgo.Session, intr *discordgo.InteractionCreate, content string, 
		ephemeral bool) {
			
	flags := discordgo.MessageFlags(0)
	if ephemeral {
		flags = discordgo.MessageFlagsEphemeral
	}
	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
}

func updateMessage(session *discordgo.Session, intr *discordgo.InteractionCreate, content string,
	components []discordgo.MessageComponent) {

	if components == nil {
		components = []discordgo.MessageComponent{}
	}
	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
		},
	})
}
