package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"

	"github.com/Aleksss34/helper/backend/internal/dto"
)

var (
	commandCategoryRegex = regexp.MustCompile(`(?m)^([А-Яа-яA-Za-z0-9\s\(\),]+):$`)
)

func (p *Parser) ParseCommands(ctx context.Context) error {
	var op = "service.parser.ParseLocalCommands"
	log := p.log.With(slog.String("op", op))

	data, err := os.ReadFile(p.commandsPath)
	if err != nil {
		log.Error("Не удалось прочитать файл", slog.String("Путь", p.commandsPath))
		return fmt.Errorf("%s: не удалось прочесть файл: %w", op, err)
	}
	log.Info("Начинаем парсить команды из файла")

	rawText := string(data)
	chunks := p.chunkCommandsText(rawText, p.commandsPath)

	points := make([]*dto.Point, 0, len(chunks))
	for id, chunk := range chunks {
		log.Info("Чанк успешно запаршен", slog.Int("Номер", id+1))
		pointId := p.hashToUint64(chunk.SourceURL + "#" + strconv.Itoa(int(id)))
		point := p.getPoint(ctx, chunk, pointId, p.vocab, p.avgDL)
		points = append(points, point)

		if len(points) >= p.batchSize {
			if err := p.qdrant.Upsert(ctx, points); err != nil {
				log.Error("Не удалось сохранить чанки в Qdrant", slog.Any("error", err))
			} else {
				log.Info("Поинтеры успешно сохранены")
				_ = p.vocab.Save()
			}
			points = points[:0]
		}
	}

	if len(points) > 0 {
		if err := p.qdrant.Upsert(ctx, points); err != nil {
			log.Error("Не удалось сохранить финальные чанки в Qdrant", slog.Any("error", err))
		} else {
			log.Info("Финальные поинтеры успешно сохранены")
			_ = p.vocab.Save()
		}
	}

	return nil
}
