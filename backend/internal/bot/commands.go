package bot

import (
	"github.com/bwmarrin/discordgo"
)

var adminPerms int64 = discordgo.PermissionAdministrator

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
		Name:        "delete-my-data",
		Description: "Permanently delete your profile, timezone, and leave all standups",
	},
	{
		Name:        "timezone",
		Description: "Set your local timezone for standup reminders",
	},
	{
		Name:                     "set-channel",
		Description:              "Set where reports are posted (Admin only)",
		DefaultMemberPermissions: &adminPerms,
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
		Name:                     "create-standup",
		Description:              "Create a new team standup (Admin only)",
		DefaultMemberPermissions: &adminPerms,
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
		Name:                     "edit-standup",
		Description:              "Edit an existing standup team (Admin only)",
		DefaultMemberPermissions: &adminPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "standup_name",
				Description:  "The exact name of the standup to edit",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "new_channel",
				Description: "Change where reports should be posted",
				Required:    false,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "new_time",
				Description: "New time to trigger standup (HH:MM in 24h format, e.g. 09:30)",
				Required:    false,
			},
		},
	},
	{
		Name:                     "delete-standup",
		Description:              "Permanently delete an existing standup team (Admin only)",
		DefaultMemberPermissions: &adminPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "name",
				Description:  "Name of the standup you want to delete",
				Required:     true,
				Autocomplete: true,
			},
		},
	},
	{
		Name:                     "add-member",
		Description:              "Add a user to an existing standup (Admin Only)",
		DefaultMemberPermissions: &adminPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user you want to add",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "standup_name",
				Description:  "The exact name of the standup (e.g. 'Backend')",
				Required:     true,
				Autocomplete: true,
			},
		},
	},
	{
		Name:                     "remove-member",
		Description:              "Remove a user from an existing standup (Admin Only)",
		DefaultMemberPermissions: &adminPerms,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user you want to remove",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "standup_name",
				Description:  "The exact name of the standup (e.g. 'Backend')",
				Required:     true,
				Autocomplete: true,
			},
		},
	},
	{
		Name:        "history",
		Description: "View past standup reports",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionUser,
				Name:        "user",
				Description: "The user whose history you want to see",
				Required:    true,
			},
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "standup_name",
				Description:  "The standup team",
				Required:     true,
				Autocomplete: true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionInteger,
				Name:        "days",
				Description: "Number of days to look back (default 5, max 10)",
				Required:    false,
			},
		},
	},
}
