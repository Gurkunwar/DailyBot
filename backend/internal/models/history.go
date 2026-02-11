package models

import "gorm.io/gorm"

type StandupHistory struct {
	gorm.Model
	UserID string `gorm:"index"`
	Date   string `gorm:"index"`
}
