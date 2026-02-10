package models

import "gorm.io/gorm"

type Guild struct {
	gorm.Model
	GuildID string `gorm:"uniqueIndex"`
	ReportChannelID string
	StandupTime string 
}