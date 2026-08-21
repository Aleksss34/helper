package dto

type Point struct {
	Id        uint64
	Embedding []float32
	Server    string
	Title     string
	Content   string
	URL       string
}
