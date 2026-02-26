package api

import (
	"encoding/json"
	"fmt"
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

func (s *Server) HandleUpdateStandup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ID              uint     `json:"id"`
		Name            string   `json:"name"`
		Time            string   `json:"time"`
		ReportChannelID string   `json:"report_channel_id"`
		Questions       []string `json:"questions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)

	var standup models.Standup
	if err := s.DB.Where("id = ? AND manager_id = ?", payload.ID, managerID).First(&standup).Error; err != nil {
		http.Error(w, "Standup not found or unauthorized", http.StatusUnauthorized)
		return
	}

	standup.Name = payload.Name
	standup.Time = payload.Time
	standup.ReportChannelID = payload.ReportChannelID
	standup.Questions = payload.Questions

	if err := s.DB.Save(&standup).Error; err != nil {
		http.Error(w, "Failed to update standup", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Standup updated successfully!",
	})
}

func (s *Server) HandleGetStandupHistory(w http.ResponseWriter, r *http.Request) {
    standupID := r.URL.Query().Get("standup_id")
    if standupID == "" {
        http.Error(w, "Missing standup_id parameter", http.StatusBadRequest)
        return
    }

    var histories []models.StandupHistory
    if err := s.DB.Where("standup_id = ?", standupID).
        Order("created_at desc").
        Limit(50).
        Find(&histories).
        Error; err != nil {

        http.Error(w, "Failed to fetch history", http.StatusInternalServerError)
        return
    }

    var response []HistoryDTO
    for _, h := range histories {
        response = append(response, HistoryDTO{
            ID:        h.ID,
            UserID:    h.UserID,
            Date:      h.Date,
            Answers:   h.Answers,
            CreatedAt: h.CreatedAt.Format("2006-01-02 15:04:05"),
        })
    }

    if response == nil {
        response = []HistoryDTO{}
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleAddStandupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		StandupID uint   `json:"standup_id"`
		UserID    string `json:"user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid Payload", http.StatusBadRequest)
		return
	}

	if err := s.StandupService.AddMemberToStandup(reqBody.UserID, reqBody.StandupID); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var standup models.Standup
	s.DB.First(&standup, reqBody.StandupID)

	dmChannel, err := s.Session.UserChannelCreate(reqBody.UserID)
	if err == nil {
		welcomeMsg := fmt.Sprintf(
			"👋 **You've been added to the '%s' Standup!**\n\n"+
				"You can now submit your daily reports for this team.\n"+
				"Run `/start` here or in the server to begin.",
			standup.Name,
		)
		s.Session.ChannelMessageSend(dmChannel.ID, welcomeMsg)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Member added successfully"})
}

func (s *Server) HandleRemoveStandupMember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		StandupID uint   `json:"standup_id"`
		UserID    string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if err := s.StandupService.RemoveMemberFromStandup(reqBody.UserID, reqBody.StandupID); err != nil {
		http.Error(w, "Failed to remove member", http.StatusInternalServerError)
		return
	}

	var standup models.Standup
	s.DB.First(&standup, reqBody.StandupID)

	dmChannel, err := s.Session.UserChannelCreate(reqBody.UserID)
	if err == nil {
		goodbyeMsg := fmt.Sprintf("ℹ️ You have been removed from the **%s** standup team by the manager.",
			standup.Name)
		s.Session.ChannelMessageSend(dmChannel.ID, goodbyeMsg)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Member removed successfully"})
}

func (s *Server) HandleGetStandup(w http.ResponseWriter, r *http.Request) {
	standupID := r.URL.Query().Get("id")
	if standupID == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}

	var standup models.Standup
	if err := s.DB.Preload("Participants").First(&standup, standupID).Error; err != nil {
		http.Error(w, "Standup not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(standup)
}
