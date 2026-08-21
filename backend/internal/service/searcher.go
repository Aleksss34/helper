package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/ollama/ollama/api"
	"github.com/sashabaranov/go-openai"
)

func (s *Searcher) Search(ctx context.Context, question string, server string, out chan<- string) error {
	var op = "service.searcher.Search"
	log := s.log.With(slog.String("op", op))
	req := &api.EmbedRequest{
		Model: "bge-m3",
		Input: question,
	}
	embeddings, err := s.ollamaClient.Embed(ctx, req)
	if err != nil {
		log.Error("Не удалось получить эмбеддинг от bge-m3", slog.Any("error", err))
	}

	points, err := s.qdrant.Get(ctx, embeddings.Embeddings[0], server)
	if err != nil {
		log.Error("Не удалось получить совпадающие вектора", slog.Any("error", err))
		return fmt.Errorf("%s:%w", op, err)
	}
	possAnswer := ""
	for _, p := range points {
		///убрать
		fmt.Println(p.Server)
		possAnswer += fmt.Sprintf("%s\n\n", p.Content)
	}
	//тест мелкий промпт
	prompt := fmt.Sprintf(
		`Ответь на вопрос ТОЛЬКО по предоставленному тексту.

Вопрос: %s

Текст:
%s

Правила:
1. Используй только факты из текста. Ничего не додумывай.
2. Не используй общие знания и игнорируй нерелевантное.
3. Сохраняй точные числа, названия и факты из источника.
4. Если ответа нет в тексте, напиши ровно: "В предоставленных данных нет ответа на этот вопрос."
5. Пиши кратко, без слов "текст", "фрагмент", "источник".

Ответ:`,
		question, possAnswer,
	)
	//	prompt := fmt.Sprintf(
	//		`Ты — помощник, который отвечает на вопрос ТОЛЬКО на основе предоставленных ниже фрагментов текста (вариантов).
	//
	//Вопрос: %s
	//
	//Фрагменты (получены по эмбеддингам, могут быть частично нерелевантны):
	//%s
	//
	//Правила:
	//1. Используй информацию ТОЛЬКО из фрагментов выше. Не добавляй факты, детали, цифры или шаги, которых нет в тексте.
	//2. Игнорируй фрагменты, не относящиеся к вопросу, даже если они про смежные темы.
	//3. Если несколько фрагментов релевантны — объедини информацию из них в единый связный ответ, не дублируя и не противореча тексту.
	//4. Если ни один фрагмент не содержит ответа на вопрос — напиши: "В предоставленных данных нет ответа на этот вопрос."
	//5. Не делай предположений, не додумывай логику "по смыслу", не используй общие знания вне текста.
	//6. Формулируй ответ своими словами кратко и по делу, сохраняя точность формулировок (числа, названия, порядок действий — переноси дословно как в источнике).
	//7. Не упоминай слова "фрагмент", "вариант", "эмбеддинг" в самом ответе — ответ должен звучать как обычный связный текст для пользователя.
	//
	//Ответ:`,
	//		question, possAnswer,
	//	)
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

	//при локальных запросах
	//stream := true
	//reqOllama := &api.GenerateRequest{
	//	Model:   "qwen2.5:3b",
	//	Prompt:  prompt,
	//	Stream:  &stream,
	//	Options: map[string]interface{}{"temperature": 0},
	//}
	//if err = s.ollamaClient.Generate(ctx, reqOllama, func(r api.GenerateResponse) error {
	//	select {
	//	case <-ctx.Done():
	//		return fmt.Errorf("%s:%w", op, ctx.Err())
	//	case out <- r.Response:
	//		return nil
	//	}
	//}); err != nil {
	//	return fmt.Errorf("%s:%w", op, err)
	//
	//}
	return nil
}
