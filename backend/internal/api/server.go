package api

import (
	"log"
	"net/http"

	"github.com/Gurkunwar/dailybot/internal/services"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type Server struct {
	DB *gorm.DB
	Session *discordgo.Session
	StandupService *services.StandupService
}

func NewServer(db *gorm.DB, session *discordgo.Session, standupService *services.StandupService) *Server {
	return &Server{DB: db, Session: session, StandupService: standupService}
}

func (s *Server) Routes() {
	http.HandleFunc("/api/auth/discord", HandleDiscordLogin(s.DB))
	http.HandleFunc("/api/managed-standups", AuthMiddleware(s.HandleGetManagedStandups(s.Session)))
	http.HandleFunc("/api/standups/create", AuthMiddleware(s.HandleCreateStandup))

    http.HandleFunc("/api/user-guilds", AuthMiddleware(s.HandleGetUserGuilds))
    http.HandleFunc("/api/guild-channels", AuthMiddleware(s.HandleGetGuildChannels))
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