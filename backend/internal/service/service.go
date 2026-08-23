package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Aleksss34/helper/backend/internal/dto"
	"github.com/Aleksss34/helper/backend/pkg/bm25"
	"github.com/sashabaranov/go-openai"

	"github.com/ollama/ollama/api"
)

type Qdrant interface {
	Upsert(ctx context.Context, points []*dto.Point) error
	Get(ctx context.Context, embedding, sparseVal []float32, sparseIdx []uint32, server string) ([]*dto.Point, error)
	ExactSearch(ctx context.Context, articleNumber, articleTitle, server string) ([]*dto.Point, error)
	SearchSubArticles(ctx context.Context, articleNumber string, lawName string, server string) ([]*dto.Point, error)
	SearchByChapter(ctx context.Context, chapterNumber string, lawName string, server string) ([]*dto.Point, error)
}

type Parser struct {
	log          *slog.Logger
	httpClient   *http.Client
	ollamaClient *api.Client
	browserPath  string
	batchSize    int
	vocab        *bm25.Vocabulary
	avgDL        *bm25.AvgDocLength
	qdrant       Qdrant
}

type Searcher struct {
	log          *slog.Logger
	ollamaClient *api.Client
	vocab        *bm25.Vocabulary
	qdrant       Qdrant
	openaiClient *openai.Client
}
type Service struct {
	Parser   *Parser
	Searcher *Searcher
}

func NewParser(log *slog.Logger, httpClient *http.Client, ollamaClient *api.Client, browserPath string, qdrant Qdrant, batchSize int, vocab *bm25.Vocabulary, avgDL *bm25.AvgDocLength) *Parser {

	return &Parser{log: log, httpClient: httpClient, ollamaClient: ollamaClient, browserPath: browserPath, qdrant: qdrant, batchSize: batchSize, vocab: vocab, avgDL: avgDL}
}
func NewSearcher(log *slog.Logger, qdrant Qdrant, ollamaClient *api.Client, openaiClient *openai.Client, vocab *bm25.Vocabulary) *Searcher {

	return &Searcher{log: log, qdrant: qdrant, ollamaClient: ollamaClient, openaiClient: openaiClient, vocab: vocab}
}

func NewService(parser *Parser, searcher *Searcher) *Service {
	return &Service{Parser: parser, Searcher: searcher}
}
