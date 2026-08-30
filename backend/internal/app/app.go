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
	"github.com/Aleksss34/helper/backend/internal/storage/postgres"
	"github.com/Aleksss34/helper/backend/internal/storage/redis"
	"github.com/Aleksss34/helper/backend/pkg/bm25"
	hash "github.com/Aleksss34/helper/backend/pkg/hasher"

	"github.com/sashabaranov/go-openai"

	"github.com/ollama/ollama/api"
	"github.com/qdrant/go-client/qdrant"
)

type App struct {
	Server *restapp.App
}

func New(ctx context.Context, log *slog.Logger, paramsPostgres domain.PostgresParams, paramsRedis domain.RedisParams, gatewayCfg config.GatewayConfig, parserCfg config.ParserConfig, qdrantCfg config.QdrantConfig, apiKey string, timeoutServer, timeoutRefresh int64, hmacSecret string, costHasher int) *App {
	db, err := postgres.Conn(paramsPostgres)
	rdb, err := redis.Conn(ctx, paramsRedis)
	qdrantConf := &qdrant.Config{Port: qdrantCfg.Port, Host: qdrantCfg.Host}
	clientQdrant, err := qdrant.NewClient(qdrantConf)
	if err != nil {
		panic("Failed connection with Qdrant, error: " + err.Error())
	}
	ensureCollectionExists(ctx, clientQdrant, qdrantCfg.NameCollection)
	qdr := storage.NewQdrant(qdrantCfg.NameCollection, clientQdrant, qdrantCfg.LimitPoints, qdrantCfg.ScoreThreshold)

	postgres := storage.NewPostgres(db)
	rds := storage.NewRedis(rdb, timeoutRefresh)
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

	parserService := service.NewParser(log, httpClient, ollamaClient, parserCfg.BrowserPath, qdr, qdrantCfg.BatchSize, vocab, avgDL, parserCfg.CommandsPath)
	searcherService := service.NewSearcher(log, qdr, postgres, ollamaClient, clientOpenAi, vocab)

	hasher := *hash.New(costHasher)
	dummyHash, err := hasher.Hash("basePhrase")
	if err != nil {
		panic("Хешер не работает, ошибка:" + err.Error())
	}
	authService := service.NewAuth(log, postgres, rds, hasher, hmacSecret, dummyHash)
	serv := service.NewService(parserService, searcherService, authService)

	serverApp := restapp.New(log, serv, gatewayCfg.Host, gatewayCfg.Port, timeoutServer, hmacSecret)
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
