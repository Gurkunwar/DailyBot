package api

import (
	"encoding/json"
	"net/http"

	"github.com/Gurkunwar/asyncflow/internal/models"
)

func (s *Server) HandleGetUserSettings(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(UserIDKey).(string)

	var profile models.UserProfile
	if err := s.DB.Where("user_id = ?", userID).FirstOrCreate(&profile, models.UserProfile{
		UserID:   userID,
		Timezone: "UTC",
	}).Error; err != nil {
		http.Error(w, "Failed to load profile", http.StatusInternalServerError)
		return
	}

	if profile.Timezone == "" {
		profile.Timezone = "UTC"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"timezone": profile.Timezone,
	})
}

func (s *Server) HandleUpdateUserSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(UserIDKey).(string)

	var payload struct {
		Timezone string `json:"timezone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	if err := s.DB.Model(&models.UserProfile{}).Where("user_id = ?", userID).Update("timezone", payload.Timezone).Error; err != nil {
		http.Error(w, "Failed to update timezone", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Settings updated successfully!",
	})
}