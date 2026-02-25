package api

import (
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type Server struct {
	DB *gorm.DB
	Session *discordgo.Session
}

func NewServer(db *gorm.DB, session *discordgo.Session) *Server {
	return &Server{DB: db, Session: session}
}

func (s *Server) Routes() {
	http.HandleFunc("/api/auth/discord", HandleDiscordLogin(s.DB))
	http.HandleFunc("/api/managed-standups", AuthMiddleware(s.HandleGetManagedStandups(s.Session)))
}

func (s *Server) Start(port string) {
	s.Routes()
	log.Printf("🌐 API Server running on http://localhost%s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("HTTP server crashed: %v", err)
	}
}