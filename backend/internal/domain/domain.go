package domain

type PostgresParams struct {
	Host     string
	Port     string
	User     string
	DBName   string
	Password string
	Sslmode  string
}
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
