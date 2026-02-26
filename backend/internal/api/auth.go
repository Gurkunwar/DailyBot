package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type DiscordTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func HandleDiscordLogin(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
		}

		data := url.Values{}
		data.Set("client_id", os.Getenv("DISCORD_CLIENT_ID"))
		data.Set("client_secret", os.Getenv("DISCORD_CLIENT_SECRET"))
		data.Set("grant_type", "authorization_code")
		data.Set("code", code)
		data.Set("redirect_uri", os.Getenv("DISCORD_REDIRECT_URI"))

		resp, err := http.PostForm("https://discord.com/api/oauth2/token", data)
		if err != nil || resp.StatusCode != 200 {
			var errRes map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&errRes)
			log.Printf("Discord API Error: %v Status: %d", errRes, resp.StatusCode)

			http.Error(w, "Failed to exchange token with Discord", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var tokenRes DiscordTokenResponse
		json.NewDecoder(resp.Body).Decode(&tokenRes)

		req, _ := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
		req.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

		client := &http.Client{}
		userResp, err := client.Do(req)
		if err != nil || userResp.StatusCode != 200 {
			http.Error(w, "Failed to fetch user profile", http.StatusInternalServerError)
			return
		}
		defer userResp.Body.Close()

		var discordUser DiscordUser
		json.NewDecoder(userResp.Body).Decode(&discordUser)

		var user models.UserProfile
		err = db.Where(models.UserProfile{
			UserID:   discordUser.ID,
			Username: discordUser.Username,
			Avatar:   discordUser.Avatar,
		}).Assign(models.UserProfile{
			DiscordToken: tokenRes.AccessToken,
		}).FirstOrCreate(&user).Error

		if err != nil {
			// Handle database error
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id":  discordUser.ID,
			"username": discordUser.Username,
			"avatar":   discordUser.Avatar,
			"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(),
		})

		tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		if err != nil {
			http.Error(w, "Failed to generate session token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": tokenString,
			"user":  discordUser,
		})
	}
}

func (s *Server) HandleGetMe(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(UserIDKey).(string)

    var user models.UserProfile
    if err := s.DB.Where("user_id = ?", userID).First(&user).Error; err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    userDTO := MemberDTO{
        ID:       user.UserID,
        Username: user.Username,
        Avatar:   user.Avatar,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(userDTO)
}