package app

import (
	"context"
	restapp "gateway/backend/internal/app/rest"
	"gateway/backend/internal/config"
	"gateway/backend/internal/domain"
	"gateway/backend/internal/service"
	"gateway/backend/internal/storage"

	"log/slog"
	"net/http"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/qdrant/go-client/qdrant"
)

type App struct {
	Server *restapp.App
}

func New(ctx context.Context, log *slog.Logger, params domain.PostgresParams, gatewayCfg config.GatewayConfig, qdrantCfg config.QdrantConfig, timeoutServer int64) *App {
	//db, err := postgres.Conn(params)
	qdrantConf := &qdrant.Config{Port: qdrantCfg.Port, Host: qdrantCfg.Host}
	clientQdrant, err := qdrant.NewClient(qdrantConf)
	ensureCollectionExists(ctx, clientQdrant, qdrantCfg.NameCollection)
	qdr := storage.NewQdrant(qdrantCfg.NameCollection, clientQdrant, qdrantCfg.LimitPoints, qdrantCfg.ScoreThreshold)
	if err != nil {
		panic("Failed connection with Postgres, error: " + err.Error())
	}
	httpClient := &http.Client{Timeout: 40 * time.Second}
	ollamaClient, err := api.ClientFromEnvironment()
	if err != nil {
		panic("Не удалось полключиться к олламе, ошибка: " + err.Error())
	}
	parserService := service.NewParser(log, httpClient, ollamaClient, qdr, qdrantCfg.BatchSize)
	searcherService := service.NewSearcher(log, qdr, ollamaClient)
	serv := service.NewService(parserService, searcherService)

	serverApp := restapp.New(log, serv, gatewayCfg.Host, gatewayCfg.Port, timeoutServer)
	return &App{Server: serverApp}
}

func ensureCollectionExists(ctx context.Context, client *qdrant.Client, collectionName string) {
	ok, err := client.CollectionExists(ctx, collectionName)
	if err != nil {
		panic("Не удалось проверить существование коллекции qdrant, ошибка:" + err.Error())
	}
	if ok {
		return
	}
	if err = client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     1024,
			Distance: qdrant.Distance_Cosine},
		),
	},
	); err != nil {
		panic("Не удалось создать коллекцию, ошибка: " + err.Error())
	}
}
