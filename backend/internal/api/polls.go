package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Gurkunwar/asyncflow/internal/api/dtos"
	"github.com/Gurkunwar/asyncflow/internal/models"
	"github.com/bwmarrin/discordgo"
)

func (s *Server) HandleGetManagedPolls(w http.ResponseWriter, r *http.Request) {
	managerID := r.Context().Value(UserIDKey).(string)

	var polls []models.Poll
	if err := s.DB.Where("creator_id = ?", managerID).Order("id desc").Find(&polls).Error; err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var response []dtos.PollDTO
	for _, p := range polls {
		gName, cName := s.GetDiscordMetadata(p.GuildID, p.ChannelID)

		response = append(response, dtos.PollDTO{
			ID:          p.ID,
			Question:    p.Question,
			GuildName:   gName,
			ChannelName: cName,
			IsActive:    p.IsActive,
		})
	}

	if response == nil {
		response = []dtos.PollDTO{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) HandleGetPoll(w http.ResponseWriter, r *http.Request) {
	pollID := r.URL.Query().Get("id")
	if pollID == "" {
		http.Error(w, "Missing poll id", http.StatusBadRequest)
		return
	}

	var poll models.Poll
	// We ONLY preload the Options from the DB. We will get Votes live from Discord.
	if err := s.DB.Preload("Options").First(&poll, pollID).Error; err != nil {
		http.Error(w, "Poll not found", http.StatusNotFound)
		return
	}

	// Fetch the live poll state directly from the Discord channel
	msg, err := s.Session.ChannelMessage(poll.ChannelID, poll.MessageID)
	if err == nil && msg.Poll != nil {
		
		// Create a quick lookup map to match Discord's answer text to our DB Option IDs
		optMap := make(map[string]uint)
		for _, o := range poll.Options {
			optMap[o.Label] = o.ID
		}

		var liveVotes []models.PollVote
		
		// Loop through Discord's live answers
		for _, answer := range msg.Poll.Answers {
			optID, exists := optMap[answer.Media.Text]
			if !exists {
				continue
			}

			// Ask Discord exactly who voted for this specific answer
			voters, _ := s.Session.PollAnswerVoters(poll.ChannelID, poll.MessageID, answer.AnswerID)
			for _, voter := range voters {
				liveVotes = append(liveVotes, models.PollVote{
					PollID:   poll.ID,
					OptionID: optID,
					UserID:   voter.ID,
				})
			}
		}
		
		// Inject the live Discord votes into our response payload!
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
		Duration  int      `json:"duration"` // In hours (1, 4, 8, 24, 72, 168)
		Options   []string `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	managerID := r.Context().Value(UserIDKey).(string)

	// 1. Prepare Native Discord Poll Answers
	var pollAnswers []discordgo.PollAnswer
	for _, optText := range payload.Options {
		pollAnswers = append(pollAnswers, discordgo.PollAnswer{
			Media: &discordgo.PollMedia{
				Text: optText,
			},
		})
	}

	// 2. Construct the Native Poll
	nativePoll := &discordgo.Poll{
		Question: discordgo.PollMedia{
			Text: payload.Question,
		},
		Answers:          pollAnswers,
		AllowMultiselect: false,
		Duration:         payload.Duration,
	}

	// 3. Publish to Discord!
	msg, err := s.Session.ChannelMessageSendComplex(payload.ChannelID, &discordgo.MessageSend{
		Poll: nativePoll,
	})
	if err != nil {
		http.Error(w, "Failed to publish poll to Discord", http.StatusInternalServerError)
		return
	}

	// 4. Save to Database
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

	// 1. Tell Discord to expire the native poll immediately
	endpoint := discordgo.EndpointChannel(poll.ChannelID) + "/polls/" + poll.MessageID + "/expire"
	_, err := s.Session.RequestWithBucketID("POST", endpoint, map[string]interface{}{},
		discordgo.EndpointChannelMessage(poll.ChannelID, ""))
	if err != nil {
		log.Printf("Failed to end poll on Discord: %v", err)
		// We won't block the DB update if the Discord message was already deleted
	}

	// 2. Update our database
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
	// Verify the poll exists and belongs to this manager
	if err := s.DB.Where("id = ? AND creator_id = ?", pollID, managerID).First(&poll).Error; err != nil {
		http.Error(w, "Poll not found or unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch live data directly from Discord
	msg, err := s.Session.ChannelMessage(poll.ChannelID, poll.MessageID)
	if err != nil || msg.Poll == nil {
		http.Error(w, "Could not fetch live poll from Discord", http.StatusInternalServerError)
		return
	}

	// Build the CSV string (Reusing your exact logic from poll_commands.go!)
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

	// Tell the browser to download this response as a CSV file
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=poll_%s_results.csv", pollID))
	
	w.Write([]byte(csvBuilder.String()))
}