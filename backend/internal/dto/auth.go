package dto

type RegisterRequest struct {
	Username       string `json:"username"`
	Email          string `json:"email"`
	Password       string `json:"password"`
	RepeatPassword string `json:"repeat_password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type RefreshRequest struct {
	RefreshToken string `json:"refresh-token"`
}

type HistoryRequest struct {
	Id       int64  `json:"id"`
	Request  string `json:"request"`
	Response string `json:"response"`
}

type NewNameRequest struct {
	NewName string `json:"new_name"`
}

type UserInfo struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}
