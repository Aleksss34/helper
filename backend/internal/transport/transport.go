package transport

import (
	"context"
	"log/slog"

	"github.com/Aleksss34/helper/backend/internal/domain"
)

type SearcherService interface {
	Search(ctx context.Context, question string, server string, out chan<- string) error
}
type ParserService interface {
	ParseWiki(ctx context.Context) error
	ParseLegislation(ctx context.Context) error
	ParseRules(ctx context.Context) error
	ParseCommands(ctx context.Context) error
}

type AuthService interface {
	RegisterService(ctx context.Context, username, email, password string) (string, string, error)
	LoginService(ctx context.Context, username, pass string) (string, string, error)
	LogoutService(ctx context.Context, refreshToken string) error
	RefreshService(ctx context.Context, refreshToken string) (string, string, error)
	IsAdminService(ctx context.Context, userId int64) (bool, error)
	MeService(ctx context.Context) (*domain.User, error)
}
type Parser struct {
	log  *slog.Logger
	serv ParserService
}
type Searcher struct {
	log  *slog.Logger
	serv SearcherService
}

type Auth struct {
	log        *slog.Logger
	serv       AuthService
	hmacSecret string
}
type Transport struct {
	parser   *Parser
	searcher *Searcher
	auth     *Auth
}

func NewParser(log *slog.Logger, serv ParserService) *Parser {
	return &Parser{
		serv: serv,
		log:  log,
	}
}
func NewSearcher(log *slog.Logger, serv SearcherService) *Searcher {
	return &Searcher{
		serv: serv,
		log:  log,
	}
}
func NewAuth(log *slog.Logger, serv AuthService, hmacSecret string) *Auth {
	return &Auth{log: log, serv: serv, hmacSecret: hmacSecret}
}
func NewTransport(parser *Parser, searcher *Searcher, auth *Auth) *Transport {
	return &Transport{parser: parser, searcher: searcher, auth: auth}
}
