package dtos

type StandupDTO struct {
	ID              uint   `json:"id"`
	Name            string `json:"name"`
	Time            string `json:"time"`
	GuildName       string `json:"guild_name"`
	ChannelName     string `json:"channel_name"`
	ReportChannelID string `json:"report_channel_id"`
}