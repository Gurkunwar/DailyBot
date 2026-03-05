package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/Gurkunwar/asyncflow/internal/api/dtos"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (s *Server) HandleGetManagedStandups(dg *discordgo.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		managerID := r.Context().Value(UserIDKey).(string)
		onlyMe := r.URL.Query().Get("filter") == "me"

		// 1. Pagination & Search Setup
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit <= 0 {
			limit = 12
		}
		offset := (page - 1) * limit

		searchQuery := r.URL.Query().Get("search")

		// 2. Pre-fetch Admin Guilds for optimization (Stops N+1 Discord API limits)
		var adminGuildIDs []string
		userGuilds, err := dg.UserGuilds(100, "", "", false)
		if err == nil {
			for _, g := range userGuilds {
				if g.Owner ||
					g.Permissions&discordgo.PermissionAdministrator != 0 ||
					g.Permissions&discordgo.PermissionManageGuild != 0 {
					adminGuildIDs = append(adminGuildIDs, g.ID)
				}
			}
		}

		// 3. Construct Query
		query := s.DB.Model(&models.Standup{}).Order("id desc")

		if onlyMe {
			query = query.Where("manager_id = ?", managerID)
		} else {
			if len(adminGuildIDs) > 0 {
				query = query.Where("manager_id = ? OR guild_id IN ?", managerID, adminGuildIDs)
			} else {
				query = query.Where("manager_id = ?", managerID)
			}
		}

		// Apply Search Filter
		if searchQuery != "" {
			query = query.Where("name ILIKE ?", "%"+searchQuery+"%")
		}

		// 4. Count and Fetch
		var totalCount int64
		query.Count(&totalCount)

		var allStandups []models.Standup
		if err := query.Offset(offset).Limit(limit).Find(&allStandups).Error; err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// 5. Map to Response DTO
		var response []dtos.StandupDTO
		for _, st := range allStandups {
			// Re-using your optimized GetDiscordMetadata helper!
			gName, cName := s.GetDiscordMetadata(st.GuildID, st.ReportChannelID)

			response = append(response, dtos.StandupDTO{
				ID:              st.ID,
				Name:            st.Name,
				Time:            st.Time,
				GuildName:       gName,
				ChannelName:     cName,
				ReportChannelID: st.ReportChannelID,
			})
		}

		if response == nil {
			response = []dtos.StandupDTO{}
		}
		totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data":        response,
			"total_count": totalCount,
			"page":        page,
			"total_pages": totalPages,
		})
	}
}

func (s *Server) HandleCreateStandup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Name            string   `json:"name"`
		Time            string   `json:"time"`
		Days            string   `json:"days"`
		GuildID         string   `json:"guild_id"`
		ReportChannelID string   `json:"report_channel_id"`
		Questions       []string `json:"questions"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)

	standup := models.Standup{
		Name:            payload.Name,
		Time:            payload.Time,
		Days:            payload.Days,
		GuildID:         payload.GuildID,
		ReportChannelID: payload.ReportChannelID,
		ManagerID:       managerID,
		Questions:       payload.Questions,
	}

	if err := s.StandupService.CreateStandup(standup); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.DB.Where("guild_id = ? AND name = ?", standup.GuildID, standup.Name).First(&standup)

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
		Days            string   `json:"days"`
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
	standup.Days = payload.Days
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

func (s *Server) HandleDeleteStandup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	standupID := r.URL.Query().Get("id")
	if standupID == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)

	var standup models.Standup
	if err := s.DB.Where("id = ? AND manager_id = ?", standupID, managerID).First(&standup).Error; err != nil {
		http.Error(w, "Standup not found or unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.DB.Model(&standup).Association("Participants").Clear(); err != nil {
		log.Println("Error clearing standup participants during deletion:", err)
	}

	if err := s.DB.Unscoped().Delete(&standup).Error; err != nil {
		http.Error(w, "Failed to delete standup", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Standup deleted successfully!",
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
