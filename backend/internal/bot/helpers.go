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

func formatLocalTime(utcTimeStr string, userTZ string) string {
	if userTZ == "" {
		return fmt.Sprintf("**%s UTC**\n> *Tip: Run `/timezone` to get reminders in your local time!*", utcTimeStr)
	}

	loc, err := time.LoadLocation(userTZ)
	if err != nil {
		return fmt.Sprintf("**%s UTC**\n> *Tip: Run `/timezone` to get reminders in your local time!*", utcTimeStr)
	}

	parsedTime, err := time.Parse("15:04", utcTimeStr)
	if err != nil {
		return fmt.Sprintf("**%s** (%s)", utcTimeStr, userTZ)
	}

	now := time.Now().UTC()
	utcDate := time.Date(now.Year(), now.Month(), now.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, time.UTC)
	localDate := utcDate.In(loc)

	return fmt.Sprintf("**%s** (%s)", localDate.Format("15:04"), userTZ)
}