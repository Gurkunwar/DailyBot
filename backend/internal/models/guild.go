package models

import "gorm.io/gorm"

type Guild struct {
	gorm.Model
	GuildID string `gorm:"uniqueIndex"`
	Standups []Standup `gorm:"foreignKey:GuildID;references:GuildID"`
}