package models

import (
	"gorm.io/gorm"
)

type StandupHistory struct {
	gorm.Model
	UserID    string         `gorm:"index" json:"user_id"`
	StandupID uint           `gorm:"index" json:"standup_id"`
	Date      string         `gorm:"index" json:"date"`
	Answers   []string `gorm:"type:text;serializer:json" json:"answers"`
}