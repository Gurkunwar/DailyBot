package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/Gurkunwar/asyncflow/internal/api/dtos"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (s *Server) HandleGetManagedPolls(w http.ResponseWriter, r *http.Request) {
    managerID := r.Context().Value(UserIDKey).(string)
    onlyMe := r.URL.Query().Get("filter") == "me"

    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 { page = 1 }
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit <= 0 { limit = 12 }
    offset := (page - 1) * limit

    searchQuery := r.URL.Query().Get("search")
    guildFilter := r.URL.Query().Get("guild_id")

    var combos []struct {
        GuildID   string
        ChannelID string
    }
    s.DB.Model(&models.Poll{}).Distinct("guild_id", "channel_id").Select("guild_id", "channel_id").Find(&combos)

    var adminGuildIDs []string
    for _, gc := range combos {
        if gc.ChannelID == "" { continue }
        p, err := s.Session.UserChannelPermissions(managerID, gc.ChannelID)
        if err == nil && (p&discordgo.PermissionAdministrator != 0 || p&discordgo.PermissionManageGuild != 0 || p&discordgo.PermissionManageServer != 0) {
            adminGuildIDs = append(adminGuildIDs, gc.GuildID)
        }
    }

    query := s.DB.Model(&models.Poll{}).Order("id desc")

    if guildFilter != "" && guildFilter != "All" {
        query = query.Where("guild_id = ?", guildFilter)
        
        if onlyMe {
            query = query.Where("creator_id = ?", managerID)
        } else {
            isAdminOfSelected := false
            for _, id := range adminGuildIDs {
                if id == guildFilter {
                    isAdminOfSelected = true
                    break
                }
            }
            
            if !isAdminOfSelected {
                query = query.Where("creator_id = ?", managerID)
            }
        }
    } else {
        if onlyMe {
            query = query.Where("creator_id = ?", managerID)
        } else {
            if len(adminGuildIDs) > 0 {
                query = query.Where(
                    s.DB.Where("creator_id = ?", managerID).Or("guild_id IN ?", adminGuildIDs),
                )
            } else {
                query = query.Where("creator_id = ?", managerID)
            }
        }
    }

    if searchQuery != "" {
        query = query.Where("question ILIKE ?", "%"+searchQuery+"%")
    }

    var totalCount int64
    query.Count(&totalCount)

    var allPolls []models.Poll
    if err := query.Offset(offset).Limit(limit).Find(&allPolls).Error; err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    var response []dtos.PollDTO
    for _, p := range allPolls {
        gName, cName := s.GetDiscordMetadata(p.GuildID, p.ChannelID)
        response = append(response, dtos.PollDTO{
            ID:          p.ID,
            Question:    p.Question,
            GuildName:   gName,
            ChannelName: cName,
            IsActive:    p.IsActive,
        })
    }
    if response == nil { response = []dtos.PollDTO{} }

    totalPages := int(math.Ceil(float64(totalCount) / float64(limit)))

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "data":        response,
        "total_count": totalCount,
        "page":        page,
        "total_pages": totalPages,
    })
}

func (s *Server) HandleGetPoll(w http.ResponseWriter, r *http.Request) {
	pollID := r.URL.Query().Get("id")
	if pollID == "" {
		http.Error(w, "Missing poll id", http.StatusBadRequest)
		return
	}

	var poll models.Poll
	if err := s.DB.Preload("Options").First(&poll, pollID).Error; err != nil {
		http.Error(w, "Poll not found", http.StatusNotFound)
		return
	}

	msg, err := s.Session.ChannelMessage(poll.ChannelID, poll.MessageID)
	if err == nil && msg.Poll != nil {
		optMap := make(map[string]uint)
		for _, o := range poll.Options {
			optMap[o.Label] = o.ID
		}

		var liveVotes []models.PollVote
		for _, answer := range msg.Poll.Answers {
			optID, exists := optMap[answer.Media.Text]
			if !exists {
				continue
			}

			voters, _ := s.Session.PollAnswerVoters(poll.ChannelID, poll.MessageID, answer.AnswerID)
			for _, voter := range voters {
				liveVotes = append(liveVotes, models.PollVote{
					PollID:   poll.ID,
					OptionID: optID,
					UserID:   voter.ID,
				})
			}
		}

		poll.Votes = liveVotes
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(poll)
}

func (s *Server) HandleCreateWebPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		GuildID   string   `json:"guild_id"`
		ChannelID string   `json:"channel_id"`
		Question  string   `json:"question"`
		Duration  int      `json:"duration"`
		Options   []string `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)

	_, err := s.PollService.CreatePoll(
		payload.GuildID,
		payload.ChannelID,
		managerID,
		payload.Question,
		payload.Options,
		payload.Duration,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Poll published successfully!"})
}

func (s *Server) HandleDeleteWebPoll(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    pollIDStr := r.URL.Query().Get("id")
    if pollIDStr == "" {
        http.Error(w, "Missing poll id", http.StatusBadRequest)
        return
    }
    
    pollID, err := strconv.ParseUint(pollIDStr, 10, 32)
    if err != nil {
        http.Error(w, "Invalid poll ID", http.StatusBadRequest)
        return
    }

    managerID := r.Context().Value(UserIDKey).(string)

    var poll models.Poll
    if err := s.DB.Where("id = ? AND creator_id = ?", pollID, managerID).First(&poll).Error; err != nil {
        http.Error(w, "Poll not found or unauthorized", http.StatusUnauthorized)
        return
    }

    if err := s.PollService.DeletePoll(uint(pollID)); err != nil {
        http.Error(w, "Failed to delete poll", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "Poll deleted successfully"})
}

func (s *Server) HandleEndWebPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PollID uint `json:"poll_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)

	var poll models.Poll
	if err := s.DB.Where("id = ? AND creator_id = ?", req.PollID, managerID).First(&poll).Error; err != nil {
		http.Error(w, "Poll not found or unauthorized", http.StatusUnauthorized)
		return
	}

	if err := s.PollService.EndPoll(req.PollID); err != nil {
        http.Error(w, "Failed to end poll", http.StatusInternalServerError)
        return
    }

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Poll ended successfully"})
}

func (s *Server) HandleExportWebPoll(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    pollIDStr := r.URL.Query().Get("id")
    if pollIDStr == "" {
        http.Error(w, "Missing poll id", http.StatusBadRequest)
        return
    }

    parsedPollID, err := strconv.ParseUint(pollIDStr, 10, 32)
    if err != nil {
        http.Error(w, "Invalid poll ID", http.StatusBadRequest)
        return
    }

    managerID := r.Context().Value(UserIDKey).(string)

    var poll models.Poll
    if err := s.DB.Where("id = ? AND creator_id = ?", parsedPollID, managerID).First(&poll).Error; err != nil {
        http.Error(w, "Poll not found or unauthorized", http.StatusUnauthorized)
        return
    }

    csvData, err := s.PollService.GenerateCSVExport(uint(parsedPollID))
    if err != nil {
        http.Error(w, "Failed to export", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=poll_%s_results.csv", pollIDStr))
    w.Write([]byte(csvData))
}