package models

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Standup struct {
	gorm.Model
	Name            string         `json:"name"`
	GuildID         string         `gorm:"index" json:"guild_id"`
	ManagerID       string         `json:"manager_id"`
	ReportChannelID string         `json:"report_channel_id"`
	Questions       pq.StringArray `gorm:"type:text[]" json:"questions"`
	Time            string         `default:"09:00" json:"time"`
	Days            string
	Participants    []UserProfile  `gorm:"many2many:standup_participants;" json:"participants"`
}
