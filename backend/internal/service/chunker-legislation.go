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
var reArticle = regexp.MustCompile(`(?i)^Статья\s+\d+(?:\.\d+)*[\.\:]?\s+`)
var reChapter = regexp.MustCompile(`(?i)^ГЛАВА\s+\d+`)
var reNumberedPart = regexp.MustCompile(`^\d+\.\s+\S`)

func (p *Parser) chunkLegislationArticle(a dto.Article) []dto.Chunk {

	var chunks []dto.Chunk
	scanner := bufio.NewScanner(strings.NewReader(a.Content))

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	currentChapter := ""
	i := 0
	for i < len(lines) {
		line := p.cleanLine(lines[i])

		if p.isIgnorableLine(line) {
			i++
			continue
		}

		// 2. Если встретили главу — обновляем контекст и идем дальше
		if reChapter.MatchString(line) {
			currentChapter = line
			i++
			continue
		}

		// 1. Лог изменений по дате
		if dateMatch := reDate.FindString(line); dateMatch != "" {
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

		// 2. Статья кодекса
		if reArticle.MatchString(line) {
			sectionTitle, text, newI := p.chunkArticleBlock(i, lines)
			if currentChapter != "" {
				sectionTitle = fmt.Sprintf("%s: %s", currentChapter, sectionTitle)
			}
			textParts := p.splitLongText(sectionTitle, text, maxChunkWords)
			for idx, part := range textParts {
				partTitle := sectionTitle
				// Если статья разбилась на несколько частей, добавляем префикс к Title
				if len(textParts) > 1 {
					partTitle = fmt.Sprintf("%s (Часть %d/%d)", sectionTitle, idx+1, len(textParts))
				}

				chunks = append(chunks, dto.Chunk{
					Server:       a.Server,
					ArticleTitle: a.Title,
					SectionTitle: partTitle,
					SourceURL:    a.URL,
					Text:         fmt.Sprintf("%s: %s:%s", a.Title, partTitle, part),
				})
			}

			i = newI
			continue
		}

		i++
	}

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

		// Проверяем автора
		if reAuthor.MatchString(currentLine) {
			foundAuthor = currentLine // Берем всю строку с автором (например "Andrey_Bastrykin" или "Andrey_Bastrykin, Andrey_Chepkasov")
			i++                       // Сдвигаем индекс за автора
			break                     // Автор найден — закрываем чанк
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

	// В качестве SectionTitle берем первую строку (заголовок статьи)

	sectionTitle := p.cleanLine(lines[startIndex])
	// Начинаем сбор со СЛЕДУЮЩЕЙ строки, чтобы не дублировать заголовок в тело
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
		reChapter.MatchString(line)
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

	// Сначала пробуем найти границы "естественных" частей статьи (1. 2. 3. ...)
	var partBoundaries []int
	for i, line := range lines {
		if reNumberedPart.MatchString(strings.TrimSpace(line)) {
			partBoundaries = append(partBoundaries, i)
		}
	}

	// Если нашли достаточно естественных границ — режем строго по ним,
	// не разрывая пронумерованные части и их подпункты
	if len(partBoundaries) >= 2 {
		return p.splitByBoundaries(lines, partBoundaries, maxWords)
	}

	// Иначе — запасной вариант
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
		// Режем ТОЛЬКО на границе новой пронумерованной части, и только
		// если уже накопили достаточно слов — не создаём микро-чанки
		// на каждую часть по отдельности
		if boundarySet[i] && currentWords >= maxWords/2 && len(current) > 0 {
			flush()
		}
		current = append(current, line)
		currentWords += len(strings.Fields(line))
	}
	flush()

	// Пост-обработка: любой кусок, всё ещё превышающий лимит (объёмная
	// пронумерованная часть сама по себе, или длинный хвост без номеров
	// после последней границы), досекается словной разбивкой
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

	// Вычисляем, на сколько РАВНЫХ частей нужно разбить статью
	// Например: 360 слов при лимите 250 -> 2 части по ~180 слов каждая
	numParts := (totalWords + maxWords - 1) / maxWords
	targetWordsPerPart := totalWords / numParts

	lines := strings.Split(text, "\n")
	var subChunks []string
	var currentLines []string
	currentWordCount := 0

	for _, line := range lines {
		lineWords := len(strings.Fields(line))

		// Если набрали целевую "половину" (или треть) и буфер не пустой,
		// и это не последняя часть — закрываем текущий чанк
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
