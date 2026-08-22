package service

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"

	"github.com/Aleksss34/helper/backend/internal/dto"
)

var reDate = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}|\d{2}\.\d{4}|\d{2}\.\d{2}\.\d{4})`)
var reAuthor = regexp.MustCompile(`^[A-Za-z]+_[A-Za-z]+(?:\s*,\s*[A-Za-z]+_[A-Za-z]+)*$`)
var reArticle = regexp.MustCompile(`(?i)^Статья\s+\d+(?:\.\d+)*[\.\:]?(\s+.*)?$`)
var reChapter = regexp.MustCompile(`(?i)^ГЛАВА\s+\d+`)
var reNumberedPart = regexp.MustCompile(`^\d+\.\s+\S`)
var reClause = regexp.MustCompile(`^\d+\.\d+\.?\s+\S`)

var reTableBlockStart = regexp.MustCompile(`^\[TABLE_BLOCK\]`)
var reTableBlockEnd = regexp.MustCompile(`^\[/TABLE_BLOCK\]`)

func (p *Parser) chunkLegislationArticle(a dto.Article) []dto.Chunk {
	var chunks []dto.Chunk
	scanner := bufio.NewScanner(strings.NewReader(a.Content))

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	currentChapter := ""
	var defaultBuffer []string

	flushDefault := func() {
		if len(defaultBuffer) == 0 {
			return
		}
		text := strings.Join(defaultBuffer, "\n")
		defaultBuffer = nil

		sectionTitle := currentChapter
		if sectionTitle == "" {
			words := strings.Fields(text)
			n := 6
			if len(words) < n {
				n = len(words)
			}
			sectionTitle = strings.Join(words[:n], " ")
			if n < len(words) {
				sectionTitle += "..."
			}
		}

		textParts := p.splitLongText(sectionTitle, text, maxChunkWords)
		for idx, part := range textParts {
			partTitle := sectionTitle
			if len(textParts) > 1 {
				partTitle = fmt.Sprintf("%s (Часть %d/%d)", sectionTitle, idx+1, len(textParts))
			}

			chunks = append(chunks, dto.Chunk{
				Server:       a.Server,
				ArticleTitle: a.Title,
				SectionTitle: partTitle,
				SourceURL:    a.URL,
				Text:         fmt.Sprintf("%s: %s:\n%s", a.Title, partTitle, part),
			})
		}
	}

	i := 0
	for i < len(lines) {
		line := p.cleanLine(lines[i])

		if p.isIgnorableLine(line) {
			i++
			continue
		}

		// 1. Глава
		if reChapter.MatchString(line) {
			flushDefault()
			currentChapter = line
			i++
			continue
		}

		// 2. Лог изменений по дате
		if dateMatch := reDate.FindString(line); dateMatch != "" {
			flushDefault()
			sectionTitle, text, newI := p.chunkChanges(dateMatch, i, lines)
			chunks = append(chunks, dto.Chunk{
				Server:       a.Server,
				ArticleTitle: a.Title,
				SectionTitle: sectionTitle,
				SourceURL:    a.URL,
				Text:         text,
			})
			i = newI
			continue
		}

		// 3. Статья кодекса
		if reArticle.MatchString(line) {
			flushDefault()
			sectionTitle, text, newI := p.chunkArticleBlock(i, lines)
			if currentChapter != "" {
				sectionTitle = fmt.Sprintf("%s: %s", currentChapter, sectionTitle)
			}
			textParts := p.splitLongText(sectionTitle, text, maxChunkWords)
			for idx, part := range textParts {
				partTitle := sectionTitle
				if len(textParts) > 1 {
					partTitle = fmt.Sprintf("%s (Часть %d/%d)", sectionTitle, idx+1, len(textParts))
				}

				chunks = append(chunks, dto.Chunk{
					Server:       a.Server,
					ArticleTitle: a.Title,
					SectionTitle: partTitle,
					SourceURL:    a.URL,
					Text:         fmt.Sprintf("%s: %s:\n%s", a.Title, partTitle, part),
				})
			}

			i = newI
			continue
		}

		// 4. Пункт ОУГ ("1.1. текст")
		if reClause.MatchString(line) {
			flushDefault()
			sectionTitle, text, newI := p.chunkClauseBlock(i, lines)
			if currentChapter != "" {
				sectionTitle = fmt.Sprintf("%s: %s", currentChapter, sectionTitle)
			}
			textParts := p.splitLongText(sectionTitle, text, maxChunkWords)
			for idx, part := range textParts {
				partTitle := sectionTitle
				if len(textParts) > 1 {
					partTitle = fmt.Sprintf("%s (Часть %d/%d)", sectionTitle, idx+1, len(textParts))
				}

				chunks = append(chunks, dto.Chunk{
					Server:       a.Server,
					ArticleTitle: a.Title,
					SectionTitle: partTitle,
					SourceURL:    a.URL,
					Text:         fmt.Sprintf("%s: %s:\n%s", a.Title, partTitle, part),
				})
			}

			i = newI
			continue
		}

		// 5. Полная строка/запись таблицы
		if reTableBlockStart.MatchString(line) {
			flushDefault()
			i++ // пропускаем [TABLE_BLOCK]

			var blockLines []string
			for i < len(lines) {
				current := p.cleanLine(lines[i])
				if reTableBlockEnd.MatchString(current) {
					i++ // пропускаем [/TABLE_BLOCK]
					break
				}
				if !p.isIgnorableLine(current) {
					blockLines = append(blockLines, current)
				}
				i++
			}

			if len(blockLines) > 0 {
				sectionTitle := currentChapter
				if sectionTitle == "" {
					sectionTitle = a.Title
				}

				body := strings.Join(blockLines, "\n")
				textParts := p.splitLongText(sectionTitle, body, maxChunkWords)

				for idx, part := range textParts {
					partTitle := sectionTitle
					if len(textParts) > 1 {
						partTitle = fmt.Sprintf("%s (Часть %d/%d)", sectionTitle, idx+1, len(textParts))
					}

					chunks = append(chunks, dto.Chunk{
						Server:       a.Server,
						ArticleTitle: a.Title,
						SectionTitle: partTitle,
						SourceURL:    a.URL,
						Text:         part,
					})
				}
			}
			continue
		}

		// 6. Обычные подзаголовки/контекст вне стандартных правил
		defaultBuffer = append(defaultBuffer, line)
		i++
	}

	flushDefault()
	return chunks
}

func (p *Parser) chunkChanges(dateMatch string, startIndex int, lines []string) (string, string, int) {
	startDate := dateMatch
	foundAuthor := ""
	var chunkLines []string

	i := startIndex
	for i < len(lines) {
		currentLine := p.cleanLine(lines[i])

		if i > startIndex && p.isStartOfNewChunk(currentLine) {
			break
		}

		if !p.isIgnorableLine(currentLine) {
			chunkLines = append(chunkLines, currentLine)
		}

		if reAuthor.MatchString(currentLine) {
			foundAuthor = currentLine
			i++
			break
		}

		i++
	}

	if foundAuthor == "" {
		foundAuthor = "unknown"
	}

	sectionTitle := fmt.Sprintf("Дата: %s, Автор: %s", startDate, foundAuthor)
	text := strings.Join(chunkLines, "\n")

	return sectionTitle, text, i
}

func (p *Parser) chunkArticleBlock(startIndex int, lines []string) (string, string, int) {
	var blockLines []string
	sectionTitle := p.cleanLine(lines[startIndex])
	i := startIndex + 1

	for i < len(lines) {
		line := p.cleanLine(lines[i])

		if p.isStartOfNewChunk(line) {
			break
		}

		if !p.isIgnorableLine(line) {
			blockLines = append(blockLines, line)
		}
		i++
	}

	text := strings.Join(blockLines, "\n")
	return sectionTitle, text, i
}

var reClauseNumber = regexp.MustCompile(`^\d+\.\d+\.?`)

func (p *Parser) chunkClauseBlock(startIndex int, lines []string) (string, string, int) {
	firstLine := p.cleanLine(lines[startIndex])

	sectionTitle := reClauseNumber.FindString(firstLine)
	if sectionTitle == "" {
		sectionTitle = firstLine
	}

	blockLines := []string{firstLine}
	i := startIndex + 1

	for i < len(lines) {
		line := p.cleanLine(lines[i])

		if p.isStartOfNewChunk(line) {
			break
		}

		if !p.isIgnorableLine(line) {
			blockLines = append(blockLines, line)
		}
		i++
	}

	text := strings.Join(blockLines, "\n")
	return sectionTitle, text, i
}

func (p *Parser) isStartOfNewChunk(line string) bool {
	return reDate.MatchString(line) ||
		reArticle.MatchString(line) ||
		reChapter.MatchString(line) ||
		reClause.MatchString(line) ||
		reTableBlockStart.MatchString(line)
}

func (p *Parser) isIgnorableLine(line string) bool {
	cleaned := p.cleanLine(line)
	if cleaned == "" {
		return true
	}

	lower := strings.ToLower(line)
	if lower == "закреплено" || strings.Contains(lower, "закреплено") ||
		strings.Contains(lower, "нажмите, чтобы") ||
		strings.Contains(lower, "спойлер:") ||
		strings.HasPrefix(lower, "оос") {
		return true
	}

	return false
}

func (p *Parser) cleanLine(s string) string {
	s = strings.Trim(s, " \t\r\n\u200b\ufeff\u00a0")
	return strings.TrimSpace(s)
}

const maxChunkWords = 250

func (p *Parser) splitLongText(sectionTitle string, text string, maxWords int) []string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return []string{text}
	}

	lines := strings.Split(text, "\n")
	var partBoundaries []int

	for i, line := range lines {
		if reNumberedPart.MatchString(strings.TrimSpace(line)) {
			partBoundaries = append(partBoundaries, i)
		}
	}

	if len(partBoundaries) >= 2 {
		return p.splitByBoundaries(lines, partBoundaries, maxWords)
	}

	return p.splitByWordCount(text, maxWords)
}

func (p *Parser) splitByBoundaries(lines []string, boundaries []int, maxWords int) []string {
	var rawParts []string
	var current []string
	currentWords := 0

	flush := func() {
		if len(current) > 0 {
			rawParts = append(rawParts, strings.Join(current, "\n"))
			current = nil
			currentWords = 0
		}
	}

	boundarySet := make(map[int]bool, len(boundaries))
	for _, b := range boundaries {
		boundarySet[b] = true
	}

	for i, line := range lines {
		if boundarySet[i] && currentWords >= maxWords/2 && len(current) > 0 {
			flush()
		}
		current = append(current, line)
		currentWords += len(strings.Fields(line))
	}
	flush()

	var finalParts []string
	for _, part := range rawParts {
		if len(strings.Fields(part)) > maxWords {
			finalParts = append(finalParts, p.splitByWordCount(part, maxWords)...)
		} else {
			finalParts = append(finalParts, part)
		}
	}

	return finalParts
}

func (p *Parser) splitByWordCount(text string, maxWords int) []string {
	words := strings.Fields(text)
	totalWords := len(words)

	numParts := (totalWords + maxWords - 1) / maxWords
	targetWordsPerPart := totalWords / numParts

	lines := strings.Split(text, "\n")
	var subChunks []string
	var currentLines []string
	currentWordCount := 0

	for _, line := range lines {
		lineWords := len(strings.Fields(line))

		if currentWordCount >= targetWordsPerPart && len(subChunks) < numParts-1 && len(currentLines) > 0 {
			subChunks = append(subChunks, strings.Join(currentLines, "\n"))
			currentLines = nil
			currentWordCount = 0
		}

		currentLines = append(currentLines, line)
		currentWordCount += lineWords
	}

	if len(currentLines) > 0 {
		subChunks = append(subChunks, strings.Join(currentLines, "\n"))
	}

	return subChunks
}
