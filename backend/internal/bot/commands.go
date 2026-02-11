package bot

import (
	"fmt"
	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
)

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "start",
		Description: "Manually trigger your daily standup form",
	},
	{
		Name:        "help",
		Description: "Show the DailyBot help menu",
	},
	{
		Name:        "reset",
		Description: "Clear your timezone and primary server settings",
	},
	{
		Name:        "set-channel",
		Description: "Set where reports are posted (Admin only)",
		// DefaultMemberPermissions: func() *int64 { i := int64(discordgo.PermissionAdministrator); return &i }(),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "channel",
				Description: "The channel to send reports to",
				Required:    true,
			},	
		},
	},
}

func (h *BotHanlder) handleSetChannel(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetChannelID := options[0].Value.(string)

	var guild models.Guild
	h.DB.Where(&models.Guild{GuildID: intr.GuildID}).FirstOrCreate(&guild)
	guild.ReportChannelID = targetChannelID
	h.DB.Save(&guild)

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: fmt.Sprintf("✅ Success! Daily reports will now be sent to <#%s>", targetChannelID),
        },
    })
}

func (h *BotHanlder) handleHelp(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	helpText := "💡 **DailyBot Help Menu**\n\n" +
		"`/start` - Manually trigger your daily standup form.\n" +
		"`/reset` - Clear your timezone and primary server settings.\n" +
		"`/set-channel #channel` - (Admin only) Set where reports are posted.\n\n" +
		"Note: I will automatically ping you at 9:00 AM in your saved timezone!"

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: helpText,
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
}

func (h *BotHanlder) handleReset(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	userID := intr.User.ID
    if intr.Member != nil {
        userID = intr.Member.User.ID
    }
	
	result := h.DB.Where("user_id = ?", userID).Delete(&models.UserProfile{})

	if result.Error != nil {
		session.ChannelMessageSend(intr.ChannelID, "❌ Failed to reset profile. Please try again.")
		return
	}

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "✅ Profile reset! Use `/start` to set your new timezone.",
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
}
