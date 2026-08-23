package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"unicode"

	"github.com/Aleksss34/helper/backend/internal/dto"
	targeted_search "github.com/Aleksss34/helper/backend/internal/service/targeted-search"
	"github.com/Aleksss34/helper/backend/pkg/bm25"
	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"
)

func (s *Searcher) Search(ctx context.Context, question string, server string, out chan<- string) error {
	var op = "service.searcher.Search"
	log := s.log.With(slog.String("op", op))

	points, err := s.retrievePoints(ctx, question, server, log)
	if err != nil {
		return fmt.Errorf("%s:%w", op, err)
	}

	possAnswer := ""
	for _, p := range points {
		content := s.normalizeCaps(p.Content)
		possAnswer += fmt.Sprintf("%s\n\n", content)
	}

	prompt := fmt.Sprintf(
		`Ответь на вопрос ТОЛЬКО по предоставленному тексту.
 
Вопрос: %s
 
Текст:
%s
 
Правила:
1. Используй только факты из текста. Ничего не додумывай.
2. Не используй общие знания и игнорируй нерелевантное.
3. Сохраняй точные числа, названия и факты из источника.
4. Если ответа нет в тексте, напиши ровно: "Я не знаю ответ на этот вопрос."
5. Пиши кратко, без слов "текст", "фрагмент", "источник".
6. Если в тексте есть несколько статей с одинаковым номером из разных документов, 
   НЕ выбирай одну самостоятельно. Вместо ответа напиши ровно:
   "Существует несколько статей с этим номером. Уточните, из какого документа: например, [Название документа 1] или [Название документа 2]."
   Вместо [Название документа 1] и [Название документа 2] подставь два реальных названия документов из текста, где встретилась статья с этим номером.
 
Ответ:`,
		question, possAnswer,
	)
	fmt.Println(prompt)
	stream, err := s.openaiClient.CreateChatCompletionStream(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "openai/gpt-oss-20b",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Stream:      true,
			Temperature: 0.1,
		},
	)
	if err != nil {
		log.Error("Ошибка запроса", slog.Any("error", err))
		return fmt.Errorf("%s:%w", op, err)
	}
	defer stream.Close()
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: ошибка чтения стрима: %w", op, err)
		}
		if len(resp.Choices) > 0 {
			content := resp.Choices[0].Delta.Content
			if len(content) > 0 {
				select {
				case <-ctx.Done():
					return fmt.Errorf("%s: %w", op, ctx.Err())
				case out <- content:
				}
			}
		}
	}
}
func (s *Searcher) retrievePoints(
	ctx context.Context,
	question string,
	server string,
	log *slog.Logger,
) ([]*dto.Point, error) {

	lawName := targeted_search.DetectLawName(question)

	articleNumber := targeted_search.ExtractArticleNumber(question)

	if articleNumber != "" {

		log.Info(
			"обнаружен запрос статьи",
			slog.String("article_number", articleNumber),
			slog.String("law_name", lawName),
		)

		exactPoints, err := s.qdrant.ExactSearch(
			ctx,
			articleNumber,
			lawName,
			server,
		)

		if err != nil {
			log.Warn(
				"точный поиск не удался",
				slog.Any("error", err),
				slog.String("article_number", articleNumber),
				slog.String("law_name", lawName),
			)
		} else if len(exactPoints) > 0 {

			log.Info(
				"сработал точный поиск статьи",
				slog.String("article_number", articleNumber),
				slog.String("law_name", lawName),
				slog.Int("найдено", len(exactPoints)),
			)

			return exactPoints, nil
		}

		// ищем подстатьи
		if !strings.Contains(articleNumber, ".") {

			subArticlePoints, err := s.qdrant.SearchSubArticles(
				ctx,
				articleNumber,
				lawName,
				server,
			)

			if err != nil {
				log.Warn(
					"поиск подстатей не удался",
					slog.Any("error", err),
					slog.String("article_number", articleNumber),
					slog.String("law_name", lawName),
				)
			} else if len(subArticlePoints) > 0 {

				log.Info(
					"сработал fallback на подстатьи",
					slog.String("article_number", articleNumber),
					slog.String("law_name", lawName),
					slog.Int("найдено", len(subArticlePoints)),
				)

				return subArticlePoints, nil
			}
		}

		log.Info(
			"точный поиск и поиск подстатей не дали результата, пробуем главу",
			slog.String("article_number", articleNumber),
			slog.String("law_name", lawName),
		)
	}

	// сюда попадаем, если номер статьи не найден в вопросе вообще,
	// либо если он найден, но ни точный поиск, ни подстатьи ничего не дали
	if chapterNumber := targeted_search.ExtractChapterNumber(question); chapterNumber != "" {

		log.Info(
			"обнаружен запрос главы",
			slog.String("chapter_number", chapterNumber),
			slog.String("law_name", lawName),
		)

		chapterPoints, err := s.qdrant.SearchByChapter(
			ctx,
			chapterNumber,
			lawName,
			server,
		)

		if err != nil {
			log.Warn(
				"поиск по главе не удался",
				slog.Any("error", err),
				slog.String("chapter_number", chapterNumber),
				slog.String("law_name", lawName),
			)
		} else if len(chapterPoints) > 0 {

			log.Info(
				"сработал точный поиск главы",
				slog.String("chapter_number", chapterNumber),
				slog.String("law_name", lawName),
				slog.Int("найдено", len(chapterPoints)),
			)

			return chapterPoints, nil
		}

		log.Info(
			"поиск по главе не дал результата, переходим к гибридному поиску",
			slog.String("chapter_number", chapterNumber),
			slog.String("law_name", lawName),
		)
	}

	return s.hybridSearch(ctx, question, server, log)
}

func (s *Searcher) hybridSearch(ctx context.Context, question, server string, log *slog.Logger) ([]*dto.Point, error) {
	req := &api.EmbedRequest{
		Model: "bge-m3",
		Input: question,
	}
	embeddings, err := s.ollamaClient.Embed(ctx, req)
	if err != nil {
		log.Error("Не удалось получить эмбеддинг от bge-m3", slog.Any("error", err))
		return nil, err
	}
	sparseIdx, sparseVal := bm25.QuerySparseVector(s.vocab, question)
	points, err := s.qdrant.Get(ctx, embeddings.Embeddings[0], sparseVal, sparseIdx, server)
	if err != nil {
		log.Error("Не удалось получить совпадающие вектора", slog.Any("error", err))
		return nil, err
	}
	return points, nil
}
func (s *Searcher) normalizeCaps(str string) string {
	re := regexp.MustCompile(`[А-ЯЁ]{2,}(?:\s[А-ЯЁ]{2,})*`)
	return re.ReplaceAllStringFunc(str, func(match string) string {
		words := strings.Fields(match)
		for i, w := range words {
			runes := []rune(w)

			// Аббревиатуры (короткие слова, например УК, ФСБ, ГИБДД, ОСБ, КАС, ТРК) не трогаем
			if len([]rune(w)) <= 4 {
				continue
			}

			// Определяем, стоит ли слово в начале строки или после точки
			isSentenceStart := i == 0
			if i > 0 {
				prevWord := words[i-1]
				if strings.HasSuffix(prevWord, ".") {
					isSentenceStart = true
				}
			}

			if isSentenceStart {
				lower := []rune(strings.ToLower(w))
				lower[0] = unicode.ToUpper(lower[0])
				words[i] = string(lower)
			} else {
				words[i] = strings.ToLower(w)
			}
			_ = runes
		}
		return strings.Join(words, " ")
	})
}
