package service

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/Aleksss34/helper/backend/internal/dto"
)

// Порог для обычных статей без чёткой структуры разделов
const shortArticleWordThreshold = 150

// Заголовок "Содержание" — маркер начала оглавления в тексте MediaWiki-статьи
var tocHeaderRe = regexp.MustCompile(`(?m)^Содержание\s*$`)

// Строка оглавления вида "1.2
// цифры (в т.ч. "1" или "1.2"), таб/пробелы, затем сам текст заголовка
var tocLineRe = regexp.MustCompile(`^(\d+(?:\.\d+)*)[\t ]+(.+)$`)

// listItemPattern — строка вида "Название — текст" или "Название - текст",
// характерная для разделов-перечислений (профессии, предметы, локации и т.п.)
var listItemPattern = regexp.MustCompile(`^([А-ЯA-Z][а-яa-zА-ЯA-Z0-9\s]{1,40}?)\s*[-—]\s*(.+)$`)

func (p *Parser) chunkArticle(a dto.Article) []dto.Chunk {
	wordCount := len(strings.Fields(a.Content))

	// Короткая статья — не режем
	if wordCount <= shortArticleWordThreshold {
		return []dto.Chunk{{
			ArticleTitle: a.Title,
			SourceURL:    a.URL,
			Text:         fmt.Sprintf("%s. %s", a.Title, a.Content),
		}}
	}

	sections := p.splitBySections(a.Content)

	// Не нашли структуру разделов (нет "Содержание" или заголовки не
	// удалось сопоставить) — тоже отдаём статью целиком, режем на такой
	// случай хотя бы по абзацам, чтобы не иметь один гигантский чанк
	if len(sections) <= 1 {
		return p.chunkByParagraphs(a)
	}

	var chunks []dto.Chunk
	for _, sec := range sections {
		// если раздел похож на список однотипных пунктов
		// ("Профессия - описание", "Предмет - характеристики") — режем
		// его по отдельным пунктам, а не отдаём одним большим куском.
		// Иначе вопрос про одну конкретную сущность (например, "водителя
		// автобуса") тащит в контекст текст про 9 других профессий,
		// упомянутых в том же разделе.
		if p.isListLikeSection(sec.Content) {
			chunks = append(chunks, p.chunkListSection(a.Title, sec.Heading, sec.Content, a.URL)...)
			continue
		}

		text := fmt.Sprintf("%s. %s: %s", a.Title, sec.Heading, sec.Content)
		chunks = append(chunks, dto.Chunk{
			ArticleTitle: a.Title,
			SectionTitle: sec.Heading,
			SourceURL:    a.URL,
			Text:         text,
		})
	}
	return chunks
}

// isListLikeSection проверяет, состоит ли раздел преимущественно из строк
// вида "Название - текст" (список сущностей с короткими описаниями), а не
// из связного повествовательного текста.
func (p *Parser) isListLikeSection(content string) bool {
	lines := strings.Split(content, "\n")
	listLikeCount := 0
	nonEmptyCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		nonEmptyCount++
		if listItemPattern.MatchString(line) {
			listLikeCount++
		}
	}

	if nonEmptyCount < 3 {
		// Слишком мало строк, чтобы надёжно судить — не считаем списком,
		// пусть обрабатывается как обычный раздел
		return false
	}

	// Если больше 60% непустых строк раздела подходят под паттерн
	// "Название - текст" — считаем это списком, а не прозой
	return float64(listLikeCount)/float64(nonEmptyCount) > 0.6
}

// chunkListSection режет списочный раздел по отдельным пунктам — каждая
// строка вида "Название - текст" становится своим чанком с указанием,
// к какой статье/разделу/сущности он относится.
func (p *Parser) chunkListSection(articleTitle, sectionHeading, content, url string) []dto.Chunk {
	lines := strings.Split(content, "\n")

	var chunks []dto.Chunk
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := listItemPattern.FindStringSubmatch(line)
		if m == nil {
			// Строка не подошла под паттерн (например, вводная фраза перед
			// списком) — прикрепляем её как отдельный чанк-контекст, чтобы
			// не потерять, но не пытаемся разбивать дальше
			chunks = append(chunks, dto.Chunk{
				ArticleTitle: articleTitle,
				SectionTitle: sectionHeading,
				SourceURL:    url,
				Text:         fmt.Sprintf("%s. %s: %s", articleTitle, sectionHeading, line),
			})
			continue
		}

		entityName := strings.TrimSpace(m[1])
		description := strings.TrimSpace(m[2])

		text := fmt.Sprintf("%s. %s. %s: %s", articleTitle, sectionHeading, entityName, description)
		chunks = append(chunks, dto.Chunk{
			ArticleTitle: articleTitle,
			SectionTitle: sectionHeading + " — " + entityName,
			SourceURL:    url,
			Text:         text,
		})
	}

	if len(chunks) == 0 {
		// Подстраховка: если почему-то ничего не наскреблось (весь раздел
		// оказался пустыми строками) — вернуть раздел целиком одним чанком
		return []dto.Chunk{{
			ArticleTitle: articleTitle,
			SectionTitle: sectionHeading,
			SourceURL:    url,
			Text:         fmt.Sprintf("%s. %s: %s", articleTitle, sectionHeading, content),
		}}
	}

	return chunks
}

type section struct {
	Heading string
	Content string
}

// splitBySections ищет блок "Содержание" (оглавление MediaWiki), вытаскивает
// из него список заголовков разделов, а затем режет основной текст статьи
// по этим заголовкам (они встречаются в тексте статьи повторно, уже как
// реальные подзаголовки перед соответствующим содержимым).
func (p *Parser) splitBySections(content string) []section {
	tocLoc := tocHeaderRe.FindStringIndex(content)
	if tocLoc == nil {
		return nil
	}

	// Оглавление обычно идёт сразу после заголовка "Содержание" и состоит
	// из строк вида "1.2	Текст заголовка" до первой пустой строки/конца
	// блока. Дальше начинается уже основной текст статьи, где эти же
	// заголовки встречаются самостоятельными строками (без номера).
	afterTOC := content[tocLoc[1]:]
	lines := strings.Split(afterTOC, "\n")

	var headings []string
	var bodyStartLine int
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if m := tocLineRe.FindStringSubmatch(line); m != nil {
			headings = append(headings, strings.TrimSpace(m[2]))
			continue
		}
		// Первая строка, не подошедшая под формат оглавления, — начало
		// основного текста статьи
		bodyStartLine = i
		break
	}

	if len(headings) == 0 {
		return nil
	}

	body := strings.Join(lines[bodyStartLine:], "\n")

	// Строим regex, который найдёт эти заголовки как отдельные строки
	// в основном тексте, и режем body по ним
	return p.splitBodyByHeadings(body, headings)
}

func (p *Parser) splitBodyByHeadings(body string, headings []string) []section {
	type pos struct {
		heading string
		start   int
		end     int
	}

	var positions []pos
	searchFrom := 0
	for _, h := range headings {
		// Заголовок должен встретиться как отдельная строка (с учётом
		// пробелов вокруг) — так надёжнее, чем просто strings.Index,
		// который может зацепить упоминание заголовка внутри обычного
		// предложения.
		//
		// Важно: ищем НАЧИНАЯ с конца предыдущего найденного заголовка,
		// а не с начала всего текста — иначе если текст заголовка случайно
		// повторяется раньше по статье (например, упомянут в интро или
		// другом разделе), позиции получаются не по порядку возрастания,
		// и при вычислении границ чанка (end предыдущего - start следующего)
		// получается отрицательный диапазон -> panic slice bounds.
		re := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(h) + `\s*$`)
		loc := re.FindStringIndex(body[searchFrom:])
		if loc == nil {
			continue // заголовок из оглавления не нашёлся в оставшемся тексте — пропускаем
		}
		start := searchFrom + loc[0]
		end := searchFrom + loc[1]
		positions = append(positions, pos{heading: h, start: start, end: end})
		searchFrom = end
	}

	if len(positions) == 0 {
		return nil
	}

	var sections []section

	// Текст до первого найденного заголовка — общее введение статьи,
	// тоже стоит сохранить отдельным чанком, если оно не пустое
	if intro := strings.TrimSpace(body[:positions[0].start]); intro != "" {
		sections = append(sections, section{Heading: "Введение", Content: intro})
	}

	for i, position := range positions {
		contentStart := position.end
		var contentEnd int
		if i+1 < len(positions) {
			contentEnd = positions[i+1].start
		} else {
			contentEnd = len(body)
		}
		if contentEnd < contentStart {
			p.log.Info("пропускаю раздел — некорректные границы", slog.String("Позиция", position.heading))
			continue
		}
		sectionText := strings.TrimSpace(body[contentStart:contentEnd])
		if sectionText == "" {
			continue
		}
		sections = append(sections, section{Heading: position.heading, Content: sectionText})
	}

	return sections
}

// chunkByParagraphs — запасной вариант для длинных статей без чёткой
// структуры разделов: режем просто по абзацам (двойной перенос строки),
// собирая их в чанки примерно по shortArticleWordThreshold слов.
func (p *Parser) chunkByParagraphs(a dto.Article) []dto.Chunk {
	paragraphs := strings.Split(a.Content, "\n\n")

	var chunks []dto.Chunk
	var buf strings.Builder
	wordCount := 0

	flush := func() {
		text := strings.TrimSpace(buf.String())
		if text == "" {
			return
		}
		chunks = append(chunks, dto.Chunk{
			ArticleTitle: a.Title,
			SourceURL:    a.URL,
			Text:         fmt.Sprintf("%s. %s", a.Title, text),
		})
		buf.Reset()
		wordCount = 0
	}

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		buf.WriteString(paragraph)
		buf.WriteString("\n\n")
		wordCount += len(strings.Fields(paragraph))

		if wordCount >= shortArticleWordThreshold {
			flush()
		}
	}
	flush()

	if len(chunks) == 0 {
		// Статья вообще без абзацев — отдаём как есть одним чанком
		chunks = append(chunks, dto.Chunk{
			ArticleTitle: a.Title,
			SourceURL:    a.URL,
			Text:         fmt.Sprintf("%s. %s", a.Title, a.Content),
		})
	}
	return chunks
}
