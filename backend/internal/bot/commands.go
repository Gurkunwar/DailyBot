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
	{
		Name:        "create-standup",
		Description: "Create a new team standup (Admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "name",
				Description: "Team name (e.g., Backend, Frontend)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "channel",
				Description: "Where reports should be posted",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "questions",
				Description: "Questions separated by a semicolon (;)",
				Required:    true,
			},
			{
                Type:        discordgo.ApplicationCommandOptionString,
                Name:        "members",
                Description: "Tag the members to add (e.g. @User1 @User2)",
                Required:    true,
            },
			{
                Type:        discordgo.ApplicationCommandOptionString,
                Name:        "time",
                Description: "Time to trigger standup (HH:MM in 24h format, e.g. 09:30)",
                Required:    false,
            },
		},
	},
}

func (h *BotHanlder) handleSetChannel(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	options := intr.ApplicationCommandData().Options
	targetChannelID := options[0].Value.(string)
	standupName := options[1].Value.(string)

	var standup models.Standup
    result := h.DB.Where("guild_id = ? AND name = ?", intr.GuildID, standupName).First(&standup)
    
    if result.Error != nil {
        session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "❌ Standup not found. Create it first with `/create-standup`.",
            },
        })
        return
    }

    standup.ReportChannelID = targetChannelID
    h.DB.Save(&standup)

    session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: fmt.Sprintf("✅ Reports for **%s** will now be sent to <#%s>", standup.Name, targetChannelID),
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
	var userID string
    if intr.Member != nil {
        userID = intr.Member.User.ID
    } else {
        userID = intr.User.ID
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
