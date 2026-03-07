package api

import (
	"log"
	"net/http"

	"github.com/Gurkunwar/asyncflow/internal/services"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type Server struct {
	DB             *gorm.DB
	Session        *discordgo.Session
	StandupService *services.StandupService
	PollService    *services.PollService
}

func NewServer(db *gorm.DB,
	session *discordgo.Session,
	standupService *services.StandupService,
	pollService *services.PollService) *Server {
		
	return &Server{DB: db, Session: session, StandupService: standupService, PollService: pollService}
}

func (s *Server) Routes() {
	http.HandleFunc("/", s.handleRoot)

	http.HandleFunc("/api/auth/discord", HandleDiscordLogin(s.DB))
	http.HandleFunc("/api/managed-standups", AuthMiddleware(s.HandleGetManagedStandups(s.Session)))

	http.HandleFunc("/api/dashboard/stats", AuthMiddleware(s.HandleGetDashboardStats))
	http.HandleFunc("/api/dashboard/poll-stats", AuthMiddleware(s.HandleGetPollStats))

	http.HandleFunc("/api/user-guilds", AuthMiddleware(s.HandleGetUserGuilds))
	http.HandleFunc("/api/guild-channels", AuthMiddleware(s.HandleGetGuildChannels))
	http.HandleFunc("/api/guild-members", AuthMiddleware(s.HandleGetGuildMembers))

	http.HandleFunc("/api/standups/create", AuthMiddleware(s.HandleCreateStandup))
	http.HandleFunc("/api/standups/update", AuthMiddleware(s.HandleUpdateStandup))
	http.HandleFunc("/api/standups/delete", AuthMiddleware(s.HandleDeleteStandup))
	http.HandleFunc("/api/standups/add-member", AuthMiddleware(s.HandleAddStandupMember))
	http.HandleFunc("/api/standups/remove-member", AuthMiddleware(s.HandleRemoveStandupMember))
	http.HandleFunc("/api/standups/get", AuthMiddleware(s.HandleGetStandup))
	http.HandleFunc("/api/standups/history", AuthMiddleware(s.HandleGetStandupHistory))

	http.HandleFunc("/api/managed-polls", AuthMiddleware(s.HandleGetManagedPolls))
	http.HandleFunc("/api/polls/get", AuthMiddleware(s.HandleGetPoll))
	http.HandleFunc("/api/polls/create", AuthMiddleware(s.HandleCreateWebPoll))
	http.HandleFunc("/api/polls/delete", AuthMiddleware(s.HandleDeleteWebPoll))
	http.HandleFunc("/api/polls/end", AuthMiddleware(s.HandleEndWebPoll))
	http.HandleFunc("/api/polls/export", AuthMiddleware(s.HandleExportWebPoll))
	http.HandleFunc("/api/polls/history", AuthMiddleware(s.HandleGetPollHistory))

	http.HandleFunc("/api/user/settings/get", AuthMiddleware(s.HandleGetUserSettings))
    http.HandleFunc("/api/user/settings/update", AuthMiddleware(s.HandleUpdateUserSettings))
}

func (s *Server) Start(port string) {
	s.Routes()
	log.Printf("🌐 API Server running on http://localhost%s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("HTTP server crashed: %v", err)
	}
}

func (s *Server) GetDiscordMetadata(guildID, channelID string) (string, string) {
	gName := "Unknown Server"
	if guild, err := s.Session.State.Guild(guildID); err == nil {
		gName = guild.Name
	}

	cName := "unknown-channel"
	if channel, err := s.Session.State.Channel(channelID); err == nil {
		cName = channel.Name
	}
	return gName, cName
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "online", "message": "DailyBot API is running gracefully"}`))
}
