package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/Gurkunwar/asyncflow/internal/api/dtos"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/bwmarrin/discordgo"
)

// Make sure you import "strconv" and "math" at the top of your file!

func (s *Server) HandleGetManagedPolls(w http.ResponseWriter, r *http.Request) {
    managerID := r.Context().Value(UserIDKey).(string)
    onlyMe := r.URL.Query().Get("filter") == "me"

    // 1. Pagination Setup
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 { page = 1 }
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit <= 0 { limit = 12 } // 12 fits nicely in a 3-column grid

    offset := (page - 1) * limit

	searchQuery := r.URL.Query().Get("search")

    // 2. Build Admin Guilds List for strict DB filtering
    var adminGuildIDs []string
    userGuilds, err := s.Session.UserGuilds(100, "", "", false)
    if err == nil {
        for _, g := range userGuilds {
            // Check if user has admin/management rights
            if g.Owner || g.Permissions&discordgo.PermissionAdministrator != 0 || g.Permissions&discordgo.PermissionManageServer != 0 {
                adminGuildIDs = append(adminGuildIDs, g.ID)
            }
        }
    }

    var allPolls []models.Poll
    var totalCount int64

    // 3. Construct GORM Query
    query := s.DB.Model(&models.Poll{}).Order("id desc")
    
    if onlyMe {
        query = query.Where("creator_id = ?", managerID)
    } else {
        if len(adminGuildIDs) > 0 {
            // Can see polls they created OR polls in servers they manage
            query = query.Where("creator_id = ? OR guild_id IN ?", managerID, adminGuildIDs)
        } else {
            query = query.Where("creator_id = ?", managerID)
        }
    }

	if searchQuery != "" {
        query = query.Where("question ILIKE ?", "%"+searchQuery+"%")
    }

    // 4. Count total before applying offset/limit
    query.Count(&totalCount)

    // 5. Apply pagination limit and fetch
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

    // 6. Return standard pagination payload
    resPayload := map[string]interface{}{
        "data":        response,
        "total_count": totalCount,
        "page":        page,
        "total_pages": totalPages,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(resPayload)
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

	var pollAnswers []discordgo.PollAnswer
	for _, optText := range payload.Options {
		pollAnswers = append(pollAnswers, discordgo.PollAnswer{
			Media: &discordgo.PollMedia{
				Text: optText,
			},
		})
	}

	nativePoll := &discordgo.Poll{
		Question: discordgo.PollMedia{
			Text: payload.Question,
		},
		Answers:          pollAnswers,
		AllowMultiselect: false,
		Duration:         payload.Duration,
	}

	msg, err := s.Session.ChannelMessageSendComplex(payload.ChannelID, &discordgo.MessageSend{
		Poll: nativePoll,
	})
	if err != nil {
		http.Error(w, "Failed to publish poll to Discord", http.StatusInternalServerError)
		return
	}

	pollModel := models.Poll{
		GuildID:   payload.GuildID,
		ChannelID: payload.ChannelID,
		CreatorID: managerID,
		Question:  payload.Question,
		MessageID: msg.ID,
		IsActive:  true,
	}
	s.DB.Create(&pollModel)

	for _, answerText := range payload.Options {
		s.DB.Create(&models.PollOption{
			PollID: pollModel.ID,
			Label:  answerText,
		})
	}

	receiptMessage := fmt.Sprintf("✅ Poll published! (Poll ID: `%d`)", pollModel.ID)
	s.Session.ChannelMessageSend(payload.ChannelID, receiptMessage)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Poll published successfully!"})
}

func (s *Server) HandleDeleteWebPoll(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    pollID := r.URL.Query().Get("id")
    if pollID == "" {
        http.Error(w, "Missing poll id", http.StatusBadRequest)
        return
    }

    managerID := r.Context().Value(UserIDKey).(string)
    result := s.DB.Where("id = ? AND creator_id = ?", pollID, managerID).Delete(&models.Poll{})
    
    if result.Error != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    
    if result.RowsAffected == 0 {
        http.Error(w, "Poll not found or unauthorized", http.StatusUnauthorized)
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

	endpoint := discordgo.EndpointChannel(poll.ChannelID) + "/polls/" + poll.MessageID + "/expire"
	_, err := s.Session.RequestWithBucketID("POST", endpoint, map[string]interface{}{},
		discordgo.EndpointChannelMessage(poll.ChannelID, ""))
	if err != nil {
		log.Printf("Failed to end poll on Discord: %v", err)
	}

	poll.IsActive = false
	s.DB.Save(&poll)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Poll ended successfully"})
}

func (s *Server) HandleExportWebPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	pollID := r.URL.Query().Get("id")
	if pollID == "" {
		http.Error(w, "Missing poll id", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)

	var poll models.Poll
	if err := s.DB.Where("id = ? AND creator_id = ?", pollID, managerID).First(&poll).Error; err != nil {
		http.Error(w, "Poll not found or unauthorized", http.StatusUnauthorized)
		return
	}

	msg, err := s.Session.ChannelMessage(poll.ChannelID, poll.MessageID)
	if err != nil || msg.Poll == nil {
		http.Error(w, "Could not fetch live poll from Discord", http.StatusInternalServerError)
		return
	}

	var csvBuilder strings.Builder
	csvBuilder.WriteString("Option,User ID,Username\n")

	for _, answer := range msg.Poll.Answers {
		optionText := strings.ReplaceAll(answer.Media.Text, ",", ";")

		voters, _ := s.Session.PollAnswerVoters(poll.ChannelID, poll.MessageID, answer.AnswerID)

		if len(voters) == 0 {
			csvBuilder.WriteString(fmt.Sprintf("%s,NONE,No votes\n", optionText))
		} else {
			for _, voter := range voters {
				csvBuilder.WriteString(fmt.Sprintf("%s,%s,%s\n", optionText, voter.ID, voter.Username))
			}
		}
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=poll_%s_results.csv", pollID))

	w.Write([]byte(csvBuilder.String()))
}