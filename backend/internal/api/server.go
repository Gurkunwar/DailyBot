package api

import (
	"log"
	"net/http"

	"gorm.io/gorm"
)

type Server struct {
	DB *gorm.DB
}

func NewServer(db *gorm.DB) *Server {
	return &Server{DB: db}
}

func (s *Server) Routes() {
	http.HandleFunc("/api/auth/discord", HandleDiscordLogin(s.DB))
}

func (s *Server) Start(port string) {
	s.Routes()
	log.Printf("🌐 API Server running on http://localhost%s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("HTTP server crashed: %v", err)
	}
}