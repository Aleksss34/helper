package service

import (
	"regexp"
	"strings"

	"github.com/Aleksss34/helper/backend/internal/dto"
)

func (p *Parser) chunkCommandsText(rawText string, source string) []dto.Chunk {
	lines := strings.Split(rawText, "\n")

	var chunks []dto.Chunk
	var currentCategory string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 1. Если строка является заголовком категории
		if matches := commandCategoryRegex.FindStringSubmatch(line); len(matches) > 1 {
			currentCategory = strings.TrimSpace(matches[1])
			continue
		}

		// 2. Делим строку с командой по тире/дефису ("—", "-", "–")[cite: 1]
		parts := regexp.MustCompile(`\s+[—\-–]\s+`).Split(line, 2)
		if len(parts) < 2 {
			// Резервный разделитель без отступов
			parts = regexp.MustCompile(`[—\-–]`).Split(line, 2)
		}

		if len(parts) == 2 {
			cmd := strings.TrimSpace(parts[0])
			desc := strings.TrimSpace(parts[1])

			var b strings.Builder
			if currentCategory != "" {
				b.WriteString(currentCategory)
				b.WriteString("\n")
			}
			b.WriteString("Команда: ")
			b.WriteString(cmd)
			b.WriteString("\nОписание: ")
			b.WriteString(desc)

			chunks = append(chunks, dto.Chunk{
				ArticleTitle: currentCategory,
				SectionTitle: cmd,
				SourceURL:    source,
				Text:         p.cleanChunkText(b.String()),
				Server:       "all",
			})
		}
	}

	return chunks
}
