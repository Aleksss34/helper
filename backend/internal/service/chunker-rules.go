package service

import (
	"fmt"
	"strings"

	"github.com/Aleksss34/helper/backend/internal/domain"
)

func (p *Parser) chunkRulesBlocks(pageTitle, url string, blocks []RuleBlock) []domain.Chunk {
	var chunks []domain.Chunk

	for _, block := range blocks {
		switch block.Type {
		case "table":
			// Для каждой строки таблицы делаем отдельный чанк
			for _, row := range block.Rows {
				var b strings.Builder

				// 1. Имя страницы (например, Атмосфера)
				b.WriteString(pageTitle)

				// 2. Имя блока (например, Запрещено)
				if block.Label != "" {
					b.WriteString("\n")
					b.WriteString(block.Label)
				}
				b.WriteString("\n")

				// 3. Столбцы таблицы ("Описание-...", "Наказание-..." и т.д.)
				for _, header := range block.Headers {
					value := strings.TrimSpace(row[header])
					if value == "" {
						continue
					}
					b.WriteString(header)
					b.WriteString("-")
					b.WriteString(value)
					b.WriteString("\n")
				}

				rawText := b.String()
				text := p.cleanChunkText(rawText)
				if text == "" {
					continue
				}

				chunks = append(chunks, domain.Chunk{
					ArticleTitle: pageTitle,
					SectionTitle: block.Label,
					SourceURL:    url,
					Text:         text,
					Server:       "all",
				})
			}

		case "text":
			text := strings.TrimSpace(block.Text)
			if text == "" {
				continue
			}

			parts := p.splitLongText(block.Label, text, maxRulesTextWords)

			for idx, part := range parts {
				sectionTitle := block.Label
				if len(parts) > 1 {
					sectionTitle = fmt.Sprintf("%s (Часть %d/%d)", block.Label, idx+1, len(parts))
				}

				var b strings.Builder
				b.WriteString(pageTitle)
				if block.Label != "" {
					b.WriteString("\n")
					b.WriteString(block.Label)
				}
				b.WriteString("\n")
				b.WriteString(part)

				chunks = append(chunks, domain.Chunk{
					ArticleTitle: pageTitle,
					SectionTitle: sectionTitle,
					SourceURL:    url,
					Text:         p.cleanChunkText(b.String()),
					Server:       "all",
				})
			}
		}
	}

	return chunks
}

func (p *Parser) cleanChunkText(text string) string {
	// 1. Заменяем неразрывные пробелы
	text = nbspReplacer.Replace(text)

	// 2. Очищаем каждую строку от висячих пробелов и символов • в начале
	lines := strings.Split(text, "\n")
	var cleanedLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Убираем маркер списка "• " если он мешает в начале поля
		trimmed = strings.TrimPrefix(trimmed, "•")
		trimmed = strings.TrimSpace(trimmed)

		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}

	text = strings.Join(cleanedLines, "\n")
	// 3. Убираем двойные и более переносы
	text = multiNewlineRegex.ReplaceAllString(text, "\n")

	return strings.TrimSpace(text)
}
