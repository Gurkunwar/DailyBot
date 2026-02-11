package models

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Standup struct {
	gorm.Model
	Name            string
	GuildID         string `gorm:"index"`
	ManagerID       string
	ReportChannelID string
	Questions       pq.StringArray `gorm:"type:text[]"`
}
