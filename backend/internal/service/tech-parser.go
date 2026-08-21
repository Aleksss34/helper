package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Aleksss34/helper/backend/internal/dto"
	"github.com/ollama/ollama/api"
)

func (p *Parser) getPoint(ctx context.Context, chunk dto.Chunk, id uint64) *dto.Point {
	var op = "service.tech-parser.getPoint"
	req := &api.EmbedRequest{
		Model: "bge-m3",
		Input: chunk.Text,
	}
	resp, err := p.ollamaClient.Embed(ctx, req)
	if err != nil {
		p.log.Error("Не удалось получить эмбеддинг от bge-m3", slog.Any("error", err), slog.String("op", op))
	}
	point := &dto.Point{
		Id:        id,
		Embedding: resp.Embeddings[0],
		Title:     fmt.Sprintf("%s: %s", chunk.SectionTitle, chunk.ArticleTitle),
		Content:   chunk.Text,
		URL:       chunk.SourceURL,
		Server:    chunk.Server,
	}
	return point
}
