package bot

import (
	"fmt"
	"log"

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
		Name:        "timezone",
		Description: "Set your local timezone for standup reminders",
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
			{
                Type:         discordgo.ApplicationCommandOptionString,
                Name:         "standup_name",
                Description:  "The standup to update",
                Required:     true,
                Autocomplete: true,
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
	{
		Name: "delete-standup",
		Description: "Permanently delete an existing standup team (Manager only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type: discordgo.ApplicationCommandOptionString,
				Name: "name",
				Description: "Name of the standup you want to delete",
				Required: true,
				Autocomplete: true,
			},
		},
	},
	{
		Name: "add-member",
		Description: "Add a user to an existing standup (Manager Only)",
		Options: []*discordgo.ApplicationCommandOption{
            {
                Type:        discordgo.ApplicationCommandOptionUser,
                Name:        "user",
                Description: "The user you want to add",
                Required:    true,
            }, 
            {
                Type:        discordgo.ApplicationCommandOptionString,
                Name:        "standup_name",
                Description: "The exact name of the standup (e.g. 'Backend')",
                Required:    true,
				Autocomplete: true,
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

	var user models.UserProfile
    if err := h.DB.Unscoped().Where("user_id = ?", userID).First(&user).Error; err != nil {
        session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
            Type: discordgo.InteractionResponseChannelMessageWithSource,
            Data: &discordgo.InteractionResponseData{
                Content: "❌ No profile found to reset.",
                Flags:   discordgo.MessageFlagsEphemeral,
            },
        })
        return
    }

	if err := h.DB.Model(&user).Association("Standups").Clear(); err != nil {
        log.Println("Error clearing standup teams:", err)
    }

	if result := h.DB.Unscoped().Delete(&user); result.Error != nil {
        session.ChannelMessageSend(intr.ChannelID, "❌ Failed to reset profile.")
        return
    }

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "✅ **Profile Reset Complete.** You have been removed from all standup teams.",
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
}
