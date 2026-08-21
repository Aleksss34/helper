package transport

import (
	"context"
	"log/slog"
)

type SearcherService interface {
	Search(ctx context.Context, question string, server string, out chan<- string) error
}
type ParserService interface {
	ParseWiki(ctx context.Context) error
	ParseLegislation(ctx context.Context) error
}
type Parser struct {
	log  *slog.Logger
	serv ParserService
}
type Searcher struct {
	log  *slog.Logger
	serv SearcherService
}
type Transport struct {
	parser   *Parser
	searcher *Searcher
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
func NewTransport(parser *Parser, searcher *Searcher) *Transport {
	return &Transport{parser: parser, searcher: searcher}
}
