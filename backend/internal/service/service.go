package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Aleksss34/helper/backend/internal/domain"
	"github.com/Aleksss34/helper/backend/pkg/bm25"
	hash "github.com/Aleksss34/helper/backend/pkg/hasher"
	"github.com/sashabaranov/go-openai"

	"github.com/ollama/ollama/api"
)

type Qdrant interface {
	Upsert(ctx context.Context, points []*domain.Point) error
	Get(ctx context.Context, embedding, sparseVal []float32, sparseIdx []uint32, server string) ([]*domain.Point, error)
	ExactSearch(ctx context.Context, articleNumber, articleTitle, server string) ([]*domain.Point, error)
	SearchSubArticles(ctx context.Context, articleNumber string, lawName string, server string) ([]*domain.Point, error)
	SearchByChapter(ctx context.Context, chapterNumber string, lawName string, server string) ([]*domain.Point, error)
}

type Postgres interface {
	AddUser(ctx context.Context, username, email, hashPass string) (int64, error)
	GetUser(ctx context.Context, username string) (domain.User, error)
	GetUserByID(ctx context.Context, userId int64) (domain.User, error)
	SetCountQuestions(ctx context.Context, userId int64, step int64) error
}
type Redis interface {
	AddRefresh(ctx context.Context, tokenHash string, userId int64) error
	GetIdByRefresh(ctx context.Context, tokenHash string) (string, error)
	DelRefresh(ctx context.Context, tokenHash string) error
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
	commandsPath string
}

type Searcher struct {
	log          *slog.Logger
	ollamaClient *api.Client
	vocab        *bm25.Vocabulary
	qdrant       Qdrant
	postgres     Postgres
	openaiClient *openai.Client
}

type Auth struct {
	log        *slog.Logger
	postgres   Postgres
	redis      Redis
	hasher     hash.Bcrypt
	hmacSecret string
	dummyHash  string
}
type Service struct {
	Parser   *Parser
	Searcher *Searcher
	Auth     *Auth
}

func NewParser(log *slog.Logger, httpClient *http.Client, ollamaClient *api.Client, browserPath string, qdrant Qdrant, batchSize int, vocab *bm25.Vocabulary, avgDL *bm25.AvgDocLength, commandsPath string) *Parser {

	return &Parser{log: log, httpClient: httpClient, ollamaClient: ollamaClient, browserPath: browserPath, qdrant: qdrant, batchSize: batchSize, vocab: vocab, avgDL: avgDL, commandsPath: commandsPath}
}
func NewSearcher(log *slog.Logger, qdrant Qdrant, postgres Postgres, ollamaClient *api.Client, openaiClient *openai.Client, vocab *bm25.Vocabulary) *Searcher {

	return &Searcher{log: log, qdrant: qdrant, postgres: postgres, ollamaClient: ollamaClient, openaiClient: openaiClient, vocab: vocab}
}
func NewAuth(log *slog.Logger, postgres Postgres, redis Redis, hasher hash.Bcrypt, hmacSecret string, dummyHash string) *Auth {

	return &Auth{log: log, postgres: postgres, redis: redis, hasher: hasher, hmacSecret: hmacSecret, dummyHash: dummyHash}
}
func NewService(parser *Parser, searcher *Searcher, auth *Auth) *Service {
	return &Service{Parser: parser, Searcher: searcher, Auth: auth}
}
