package models

import "gorm.io/gorm"

type StandupHistory struct {
	gorm.Model
	UserID    string `gorm:"index"`
	StandupID uint   `gorm:"index"`
	Date      string `gorm:"index"`
	Answers   []string `gorm:"type:text;serializer:json"`
}
