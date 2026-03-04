package dtos

type PollDTO struct {
	ID          uint   `json:"id"`
	Question    string `json:"question"`
	GuildName   string `json:"guild_name"`
	ChannelName string `json:"channel_name"`
	IsActive    bool   `json:"is_active"`
}