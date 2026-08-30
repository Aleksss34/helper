package restapp

import (
	"context"
	"fmt"

	"log/slog"
	"net/http"
	"time"

	"github.com/Aleksss34/helper/backend/internal/transport"

	"github.com/Aleksss34/helper/backend/internal/service"
)

type App struct {
	log    *slog.Logger
	Server *http.Server
	Port   string
}

func New(log *slog.Logger, serv *service.Service, host, port string, timeout int64, hmacSecret string) *App {
	timeoutServer := time.Duration(timeout) * time.Second
	parser := transport.NewParser(log, serv.Parser)
	searcher := transport.NewSearcher(log, serv.Searcher)
	auth := transport.NewAuth(log, serv.Auth, hmacSecret)
	trans := transport.NewTransport(parser, searcher, auth)
	router := trans.InitRouter()
	router.Use(trans.CorsMiddleware)
	addr := fmt.Sprintf("%s:%s", host, port)
	server := &http.Server{Addr: addr, Handler: router, WriteTimeout: timeoutServer, ReadTimeout: timeoutServer}
	return &App{Server: server, Port: port, log: log}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic("Failed starting server, error: " + err.Error())
	}
}
func (a *App) Run() error {
	var op = "app.rest.Run"
	a.log.Info("Server starting...", slog.String("op", op))
	if err := a.Server.ListenAndServe(); err != nil {
		return fmt.Errorf("%s:%v", op, err)
	}
	return nil
}

func (a *App) Stop() {
	var op = "app.rest.Stop"
	a.log.Info("server stopped", slog.String("op", op))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.Server.Shutdown(ctx); err != nil {
		a.log.Error("dont cancel all process, cancelling...")
		a.Server.Close()
		return
	}
	a.log.Info("server gracefully stopped")
}
