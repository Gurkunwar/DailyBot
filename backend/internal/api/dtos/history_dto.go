package dtos

type PollHistoryDTO struct {
	ID        uint   `json:"id"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	Avatar    string `json:"avatar"`
	Option    string `json:"option"`
	CreatedAt string `json:"created_at"`
}