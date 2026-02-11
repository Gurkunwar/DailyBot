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
	DB          *gorm.DB
	Session     *discordgo.Session
	TriggerFunc func(s *discordgo.Session, userID, guildID string, standupID uint)
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
	var standups []models.Standup
	s.DB.Preload("Participants").Find(&standups)

	for _, standup := range standups {
		for _, user := range standup.Participants {
			if user.Timezone == "" {
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
				result := s.DB.Where("user_id = ? AND standup_id = ? AND date = ?", user.UserID, standup.ID, today).
							First(&history)

				if result.Error != nil {
                    log.Printf("Triggering standup '%s' for user: %s", standup.Name, user.UserID)
                    s.TriggerFunc(s.Session, user.UserID, standup.GuildID, standup.ID)
                    
                    s.DB.Create(&models.StandupHistory{
                        UserID:    user.UserID,
                        StandupID: standup.ID,
                        Date:      today,
                    })
                }
			}
		}
	}
}
