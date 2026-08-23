package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Aleksss34/helper/backend/internal/dto"
	"github.com/Aleksss34/helper/pkg/bm25"
	"github.com/ollama/ollama/api"
)

func (p *Parser) getPoint(ctx context.Context, chunk dto.Chunk, id uint64, vocab *bm25.Vocabulary, avgDL *bm25.AvgDocLength) *dto.Point {
	var op = "service.tech-parser.getPoint"
	req := &api.EmbedRequest{
		Model: "bge-m3",
		Input: chunk.Text,
	}
	resp, err := p.ollamaClient.Embed(ctx, req)
	if err != nil {
		p.log.Error("Не удалось получить эмбеддинг от bge-m3", slog.Any("error", err), slog.String("op", op))
	}
	sparseIdx, sparseVal := bm25.SparseVector(vocab, avgDL, chunk.Text)
	point := &dto.Point{
		Id:            id,
		Dense:         resp.Embeddings[0],
		SparseVal:     sparseVal,
		SparseIdx:     sparseIdx,
		Title:         fmt.Sprintf("%s: %s", chunk.SectionTitle, chunk.ArticleTitle),
		Content:       chunk.Text,
		URL:           chunk.SourceURL,
		Server:        chunk.Server,
		ArticleNumber: ExtractArticleNumber(chunk.SectionTitle),
		ArticleTitle:  chunk.ArticleTitle,
		ChapterNumber: ExtractChapterNumber(chunk.SectionTitle),
	}
	return point
}
