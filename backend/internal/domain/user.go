package domain

type User struct {
	Id             int64
	Email          string
	Username       string
	HashPass       string
	Status         string
	CountQuestions int64
}
