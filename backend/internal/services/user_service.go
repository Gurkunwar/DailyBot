package services

import (
	"github.com/Gurkunwar/dailybot/internal/models"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
}

func (s *UserService) GetOrCreateProfile(userID string) (*models.UserProfile, error) {
	var profile models.UserProfile
	err := s.DB.Unscoped().Where("user_id = ?", userID).
		FirstOrCreate(&profile, models.UserProfile{UserID: userID}).Error
	if err != nil {
		return nil, err
	}

	if profile.DeletedAt.Valid {
		s.DB.Model(&profile).Unscoped().Update("deleted_at", nil)
	}
	return &profile, nil
}
