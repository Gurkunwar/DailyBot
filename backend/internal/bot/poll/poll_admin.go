package poll

import (
	"fmt"
	"strings"

	"github.com/Gurkunwar/asyncflow/internal/bot/utils"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (h *PollHandler) HandlePollEnd(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	if !utils.IsServerAdmin(intr) {
		utils.RespondWithMessage(session, intr, "⛔ This command is reserved for Server Admins.", true)
		return
	}

	pollID := uint(intr.ApplicationCommandData().Options[0].IntValue())
	if err := h.Service.EndPoll(pollID); err != nil {
        utils.RespondWithMessage(session, intr, fmt.Sprintf("❌ %v", err), true)
        return
    }

	utils.RespondWithMessage(session, intr, "✅ **Poll has been successfully closed!**", true)
}

func (h *PollHandler) HandlePollDelete(session *discordgo.Session, intr *discordgo.InteractionCreate) {
    if !utils.IsServerAdmin(intr) {
        utils.RespondWithMessage(session, intr, "⛔ This command is reserved for Server Admins.", true)
        return
    }

    pollID := uint(intr.ApplicationCommandData().Options[0].IntValue())
    
    var poll models.Poll
    if err := h.Service.DB.First(&poll, pollID).Error; err != nil {
        utils.RespondWithMessage(session, intr, "❌ Poll not found in DB.", true)
        return
    }
    
    if poll.GuildID != intr.GuildID {
        utils.RespondWithMessage(session, intr, "⛔ This poll belongs to another server.", true)
        return
    }

    if err := h.Service.DeletePoll(pollID); err != nil {
        utils.RespondWithMessage(session, intr, fmt.Sprintf("❌ %v", err), true)
        return
    }

    utils.RespondWithMessage(session, intr, "🗑️ **Poll has been successfully deleted!**", true)
}

func (h *PollHandler) HandlePollExport(session *discordgo.Session, intr *discordgo.InteractionCreate) {
    if !utils.IsServerAdmin(intr) {
        utils.RespondWithMessage(session, intr, "⛔ Admin only.", true)
        return
    }

    pollID := uint(intr.ApplicationCommandData().Options[0].IntValue())
    csvData, err := h.Service.GenerateCSVExport(pollID)
    if err != nil {
        utils.RespondWithMessage(session, intr, fmt.Sprintf("❌ %v", err), true)
        return
    }

    file := &discordgo.File{
        Name:        fmt.Sprintf("poll_results_%d.csv", pollID),
        ContentType: "text/csv",
        Reader:      strings.NewReader(csvData),
    }

    session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "📊 **Here is your Excel/CSV Export!**",
            Files:   []*discordgo.File{file},
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
}

func (h *PollHandler) handlePollList(session *discordgo.Session, intr *discordgo.InteractionCreate) {
	if !utils.IsServerAdmin(intr) {
		utils.RespondWithMessage(session, intr, "⛔ Admin only.", true)
		return
	}

	page := 1
	options := intr.ApplicationCommandData().Options
	if len(options) > 0 {
		page = int(options[0].IntValue())
		if page < 1 {
			page = 1
		}
	}

	limit := 10
	offset := (page - 1) * limit

	var recentPolls []models.Poll

	h.DB.Where("guild_id = ?", intr.GuildID).
		Order("id desc").
		Limit(limit).
		Offset(offset).
		Find(&recentPolls)

	if len(recentPolls) == 0 {
		utils.RespondWithMessage(session, intr, fmt.Sprintf("📭 No polls found on page %d.", page), true)
		return
	}

	var list strings.Builder
	list.WriteString(fmt.Sprintf("📋 **Server Polls (Page %d)**\n\n", page))

	for _, p := range recentPolls {
		qText := p.Question
		if len(qText) > 55 {
			qText = qText[:52] + "..."
		}
		list.WriteString(fmt.Sprintf("**ID: %d** | <#%s>\n> %s\n\n", p.ID, p.ChannelID, qText))
	}

	list.WriteString(fmt.Sprintf("*Use `/poll-list page: %d` to see older polls.*", page+1))

	session.InteractionRespond(intr.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: list.String(),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}