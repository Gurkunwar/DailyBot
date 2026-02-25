package api

import (
	"encoding/json"
	"net/http"

	"github.com/Gurkunwar/dailybot/internal/api/dtos"
	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (s *Server) HandleGetManagedStandups(dg *discordgo.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		managerID := r.Context().Value(UserIDKey).(string)
		var standups []models.Standup

		standups, err := s.StandupService.GetUserManagedStandups(managerID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		var response []dtos.StandupDTO

		for _, st := range standups {
			gName := "Unknown Server"

			guild, err := dg.State.Guild(st.GuildID)
			if err == nil && guild != nil {
				gName = guild.Name
			} else {
				g, err := dg.Guild(st.GuildID)
				if err == nil && g != nil {
					gName = g.Name
					dg.State.GuildAdd(g)
				}
			}

			cName := "unknown-channel"
			channel, err := dg.State.Channel(st.ReportChannelID)
			if err == nil && channel != nil {
				cName = channel.Name
			} else {
				c, err := dg.Channel(st.ReportChannelID)
				if err == nil && c != nil {
					cName = c.Name
				}
			}

			response = append(response, dtos.StandupDTO{
				ID:              st.ID,
				Name:            st.Name,
				Time:            st.Time,
				GuildName:       gName,
				ChannelName:     cName,
				ReportChannelID: st.ReportChannelID,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (s *Server) HandleCreateStandup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var standup models.Standup
	if err := json.NewDecoder(r.Body).Decode(&standup); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)
	standup.ManagerID = managerID

	if err := s.StandupService.CreateStandup(standup); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.StandupService.AddMemberToStandup(managerID, standup.ID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Standup created successfully!",
	})
}