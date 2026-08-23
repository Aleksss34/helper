package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	restapp "github.com/Aleksss34/helper/backend/internal/app/rest"
	"github.com/Aleksss34/helper/backend/internal/config"
	"github.com/Aleksss34/helper/backend/internal/domain"
	"github.com/Aleksss34/helper/backend/internal/service"
	"github.com/Aleksss34/helper/backend/internal/storage"
	"github.com/Aleksss34/helper/backend/pkg/bm25"
	"github.com/sashabaranov/go-openai"

	"github.com/ollama/ollama/api"
	"github.com/qdrant/go-client/qdrant"
)

type App struct {
	Server *restapp.App
}

func New(ctx context.Context, log *slog.Logger, params domain.PostgresParams, gatewayCfg config.GatewayConfig, parserCfg config.ParserConfig, qdrantCfg config.QdrantConfig, apiKey string, timeoutServer int64) *App {
	//db, err := postgres.Conn(params)
	qdrantConf := &qdrant.Config{Port: qdrantCfg.Port, Host: qdrantCfg.Host}
	clientQdrant, err := qdrant.NewClient(qdrantConf)
	ensureCollectionExists(ctx, clientQdrant, qdrantCfg.NameCollection)
	qdr := storage.NewQdrant(qdrantCfg.NameCollection, clientQdrant, qdrantCfg.LimitPoints, qdrantCfg.ScoreThreshold)
	if err != nil {
		panic("Failed connection with Postgres, error: " + err.Error())
	}

	if err := qdr.EnsureExactFilterIndexes(ctx); err != nil {
		panic("Не удалось создать keyboard индексы")
	}

	httpClient := &http.Client{Timeout: 40 * time.Second}
	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		panic("Не удалось полключиться к олламе, ошибка: " + err.Error())
	}
	cfgOpenai := openai.DefaultConfig(apiKey)
	cfgOpenai.BaseURL = "https://api.groq.com/openai/v1"
	clientOpenAi := openai.NewClientWithConfig(cfgOpenai)
	avgDL := bm25.NewAvgDocLength(120)
	vocab, err := bm25.LoadVocabulary("bm25_vocab.json")
	if err != nil {
		panic(err)
	}
	parserService := service.NewParser(log, httpClient, ollamaClient, parserCfg.BrowserPath, qdr, qdrantCfg.BatchSize, vocab, avgDL)
	searcherService := service.NewSearcher(log, qdr, ollamaClient, clientOpenAi, vocab)
	serv := service.NewService(parserService, searcherService)

	serverApp := restapp.New(log, serv, gatewayCfg.Host, gatewayCfg.Port, timeoutServer)
	return &App{Server: serverApp}
}

func ensureCollectionExists(ctx context.Context, client *qdrant.Client, collectionName string) {
	ok, err := client.CollectionExists(ctx, collectionName)
	if err != nil {
		panic("Не удалось проверить существование коллекции qdrant, ошибка: " + err.Error())
	}

	if ok {
		return
	}

	if err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,

		VectorsConfig: qdrant.NewVectorsConfigMap(
			map[string]*qdrant.VectorParams{
				"dense": {
					Size:     1024,
					Distance: qdrant.Distance_Cosine,
				},
			},
		),

		SparseVectorsConfig: qdrant.NewSparseVectorsConfig(
			map[string]*qdrant.SparseVectorParams{
				"sparse": {
					Modifier: qdrant.Modifier_Idf.Enum(),
				},
			},
		),
	}); err != nil {
		panic("Не удалось создать коллекцию, ошибка: " + err.Error())
	}
}
