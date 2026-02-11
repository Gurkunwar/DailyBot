package models

type StandupState struct {
	UserID      string   `json:"user_id"`
	GuildID     string   `json:"guild_id"`
	StandupID   uint     `json:"standup_id"`
	MessageID   string   `json:"message_id"`
	CurrentStep int      `json:"current_step"`
	Answers     []string `json:"answers"`
}

var StandupQuestions = []string{
	"What did you do yesterday?",
	"What are you planning to do today?",
	"Any blockers in your way?",
}
