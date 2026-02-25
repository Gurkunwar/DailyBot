package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
)

type UserGuild struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Permissions string `json:"permissions"`
	Owner       bool   `json:"owner"`
}

type GuildDTO struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BotPresent bool   `json:"bot_present"`
}

type ChannelDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Server) HandleGetUserGuilds(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(string)

	var user models.UserProfile
	if err := s.DB.Where("user_id = ?", userID).First(&user).Error; err != nil || user.DiscordToken == "" {
		http.Error(w, "User token not found", http.StatusUnauthorized)
		return
	}

	req, _ := http.NewRequest("GET", "https://discord.com/api/users/@me/guilds", nil)
	req.Header.Set("Authorization", "Bearer " + user.DiscordToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "Failed to fetch user guilds", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userGuilds []UserGuild
	json.NewDecoder(resp.Body).Decode(&userGuilds)

	responseList := make([]GuildDTO, 0)
	botGuilds := make(map[string]bool)
	
	for _, bg := range s.Session.State.Guilds {
		botGuilds[bg.ID] = true
	}

	for _, g := range userGuilds {
		perms, _ := strconv.ParseInt(g.Permissions, 10, 64)

		hasAdmin := (perms & 0x8) == 0x8
		hasManageServer := (perms & 0x20) == 0x20

		if g.Owner || hasAdmin || hasManageServer {
			responseList = append(responseList, GuildDTO{
				ID:         g.ID,
				Name:       g.Name,
				BotPresent: botGuilds[g.ID],
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseList)
}

func (s *Server) HandleGetGuildChannels(w http.ResponseWriter, r *http.Request) {
	guildID := r.URL.Query().Get("guild_id")
	if guildID == "" {
		http.Error(w, "Missing guild_id parameter", http.StatusBadRequest)
		return
	}

	channels, err := s.Session.GuildChannels(guildID)
	if err != nil {
		http.Error(w, "Failed to fetch channels", http.StatusInternalServerError)
		return
	}

	var textChannels []ChannelDTO
	for _, c := range channels {
		if c.Type == discordgo.ChannelTypeGuildText {
			textChannels = append(textChannels, ChannelDTO{
				ID:   c.ID,
				Name: c.Name,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(textChannels)
}