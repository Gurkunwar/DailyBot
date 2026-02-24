package models

type StandupState struct {
	UserID      string   `json:"user_id"`
	GuildID     string   `json:"guild_id"`
	StandupID   uint     `json:"standup_id"`
	Answers     []string `json:"answers"`
}