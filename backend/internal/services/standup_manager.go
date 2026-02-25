package services

import "github.com/Gurkunwar/dailybot/internal/models"

func (s *StandupService) AddMemberToStandup(userID string, standupID uint) error {
    var user models.UserProfile
    s.DB.Unscoped().Where("user_id = ?", userID).FirstOrCreate(&user, models.UserProfile{UserID: userID})

    if user.DeletedAt.Valid {
        s.DB.Model(&user).Unscoped().Update("deleted_at", nil)
    }

    var standup models.Standup
    if err := s.DB.First(&standup, standupID).Error; err != nil {
        return err
    }

    return s.DB.Model(&user).Association("Standups").Append(&standup)
}

func (s *StandupService) RemoveMemberFromStandup(userID string, standupID uint) error {
	var user models.UserProfile
	if err := s.DB.Unscoped().Where("user_id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	var standup models.Standup
	if err := s.DB.First(&standup, standupID).Error; err != nil {
		return err
	}

	return s.DB.Model(&user).Association("Standups").Delete(&standup)
}