package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Aleksss34/helper/backend/internal/dto"

	"github.com/ollama/ollama/api"
)

type Qdrant interface {
	Upsert(ctx context.Context, points []*dto.Point) error
	Get(ctx context.Context, embedding []float32) ([]*dto.Point, error)
}

type Parser struct {
	log          *slog.Logger
	httpClient   *http.Client
	ollamaClient *api.Client
	browserPath  string
	batchSize    int
	qdrant       Qdrant
}

type Searcher struct {
	log          *slog.Logger
	ollamaClient *api.Client
	qdrant       Qdrant
}
type Service struct {
	Parser   *Parser
	Searcher *Searcher
}

func NewParser(log *slog.Logger, httpClient *http.Client, ollamaClient *api.Client, browserPath string, qdrant Qdrant, batchSize int) *Parser {

	return &Parser{log: log, httpClient: httpClient, ollamaClient: ollamaClient, browserPath: browserPath, qdrant: qdrant, batchSize: batchSize}
}
func NewSearcher(log *slog.Logger, qdrant Qdrant, ollamaClient *api.Client) *Searcher {

	return &Searcher{log: log, qdrant: qdrant, ollamaClient: ollamaClient}
}

func NewService(parser *Parser, searcher *Searcher) *Service {
	return &Service{Parser: parser, Searcher: searcher}
}
