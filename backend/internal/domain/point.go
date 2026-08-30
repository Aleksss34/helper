package domain

type Point struct {
	Id            uint64
	Dense         []float32
	SparseIdx     []uint32
	SparseVal     []float32
	Server        string
	Title         string
	Content       string
	URL           string
	ArticleNumber string
	ArticleTitle  string
	ChapterNumber string
}
