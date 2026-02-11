package services

import (
	"log"
	"time"
	_ "time/tzdata"

	"github.com/Gurkunwar/dailybot/internal/models"
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

type StandupService struct {
	DB *gorm.DB
	Session *discordgo.Session
	TriggerFunc func(s *discordgo.Session, userID, guildID string)
}

func (s *StandupService) StartTimezoneWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			s.CheckAndTriggerStandups()
		}
	}()
}

func (s *StandupService) CheckAndTriggerStandups() {
	var users []models.UserProfile
	s.DB.Find(&users)

	for _, user := range users {
		if user.Timezone == "" || user.PrimaryGuildID == "" {
			continue
		}

		loc, err := time.LoadLocation(user.Timezone)
		if err != nil {
			continue
		}

		userLocalTime := time.Now().In(loc)
		today := userLocalTime.Format("2006-01-02")

		if userLocalTime.Hour() == 10 && userLocalTime.Minute() == 7 {
			var history models.StandupHistory
			result := s.DB.Where("user_id = ? AND date = ?", user.UserID, today).First(&history)

			if result.Error != nil {
				log.Printf("Triggering 9AM standup for user: %s", user.UserID)
				s.TriggerFunc(s.Session, user.UserID, user.PrimaryGuildID)
				s.DB.Create(&models.StandupHistory{UserID: user.UserID, Date: today})
			}
		}
	}
}