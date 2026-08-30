package dto

type UserResponse struct {
	ID             int64  `json:"id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	Status         string `json:"status"`
	CountQuestions int64  `json:"count_questions"`
}
