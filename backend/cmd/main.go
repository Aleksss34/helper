package main

import (
	"context"
	"gateway/backend/internal/app"
	"gateway/backend/internal/config"
	"gateway/backend/internal/domain"

	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"
)

const LocalEnv = "local"
const DevEnv = "dev"
const ProdEnv = "prod"

func main() {
	var op = "cmd.Main"
	ctx := context.Background()
	cfg := config.MustLoad()
	log := setupLogger(cfg.Env)
	params := domain.PostgresParams{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		DBName:   cfg.Postgres.DBName,
		Password: cfg.Postgres.Password,
		Sslmode:  cfg.Postgres.Sslmode,
	}
	application := app.New(ctx, log, params, cfg.Gateway, cfg.Qdrant, cfg.TimeoutServer)
	go func() {
		application.Server.MustRun()
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	sign := <-stop
	application.Server.Stop()
	log.Info("server stopped", slog.String("signal", sign.String()), slog.String("op", op))
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case LocalEnv:
		log = slog.New(
			tint.NewTextHandler(os.Stdout, &tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.TimeOnly,
				AddSource:  true,
			}),
		)
	case DevEnv:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)

	case ProdEnv:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}
