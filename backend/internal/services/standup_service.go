package services

import (
	"fmt"
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
	TriggerFunc func(s *discordgo.Session, userID, guildID, channelID string, standupID uint)
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

    if err := s.DB.Preload("Participants").Find(&standups).Error; err != nil {
        log.Println("Error fetching standups:", err)
        return
    }

    for _, standup := range standups {
        timeStr := standup.Time
        if timeStr == "" {
            timeStr = "09:00"
        }

		var targetHour, targetMinute int
        _, err := fmt.Sscanf(standup.Time, "%d:%d", &targetHour, &targetMinute)
        if err != nil {
            log.Printf("Invalid time format for standup %s: %s", standup.Name, standup.Time)
            continue
        }

        for _, user := range standup.Participants {
            if user.Timezone == "" {
                continue
            }

            loc, err := time.LoadLocation(user.Timezone)
            if err != nil {
                continue
            }
            userLocalTime := time.Now().In(loc)
            
            if userLocalTime.Hour() == targetHour && userLocalTime.Minute() == targetMinute {
                today := userLocalTime.Format("2006-01-02")
                var history models.StandupHistory
                result := s.DB.Where("user_id = ? AND standup_id = ? AND date = ?",
				 user.UserID, standup.ID, today).First(&history)

                if result.Error != nil {
                    log.Printf("🔔 Pinging %s for standup: %s", user.UserID, standup.Name)
                    channel, err := s.Session.UserChannelCreate(user.UserID)
                    if err == nil {
                        s.Session.ChannelMessageSend(channel.ID, 
							fmt.Sprintf("🔔 **Hey!** It's time for your **%s** standup.", standup.Name))
                        s.TriggerFunc(s.Session, user.UserID, standup.GuildID, "", standup.ID)
                    }
                }
            }
        }
    }
}