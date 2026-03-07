package models

import (
	"time"

	"gorm.io/gorm"
)

type Poll struct {
	gorm.Model
	GuildID   string
	ChannelID string
	MessageID string
	CreatorID string
	Question  string
	IsActive  bool         `gorm:"default:true"`
	Options   []PollOption `gorm:"foreignKey:PollID; constraint:OnDelete:CASCADE;"`
	Votes     []PollVote   `gorm:"foreignKey:PollID; constraint:OnDelete:CASCADE;"`
}

type PollOption struct {
	ID     uint `gorm:"primarykey"`
	PollID uint
	Label  string
}

type PollVote struct {
	ID       uint `gorm:"primarykey"`
	PollID   uint
	OptionID uint
	UserID   string
	CreatedAt time.Time
}

type PollState struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}
