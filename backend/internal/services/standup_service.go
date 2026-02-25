package services

import (
	"errors"
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

func (s *StandupService) CreateStandup(input models.Standup) error {
    if input.Name == "" {
		return errors.New("standup name cannot be empty")
	}
	if len(input.Questions) == 0 {
		return errors.New("at least one question is required")
	}

	return s.DB.Create(&input).Error
}

func (s *StandupService) GetUserManagedStandups(mangerID string) ([]models.Standup, error) {
    var standups []models.Standup
    err := s.DB.Where("manager_id = ?", mangerID).Find(&standups).Error

    return standups, err
}