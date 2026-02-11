package models

import "gorm.io/gorm"

type UserProfile struct {
	gorm.Model
	UserID string `gorm:"uniqueIndex"`
	Timezone string `default:"UTC"`
	Standups []Standup `gorm:"many2many:standup_participants;"`
}