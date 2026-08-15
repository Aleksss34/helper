package service

import (
	"context"
	"fmt"
	"gateway/backend/internal/dto"
	"log/slog"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/ollama/ollama/api"
)

const baseURL = "https://wiki.amazing-online.com"
const browserPath = "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"

type PageEntry struct {
	Title string
	URL   string
}

func (p *Parser) Parse(ctx context.Context) error {
	var op = "service.parser.Parse"
	log := p.log.With(slog.String("op", op))
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(browserPath),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	chromedbCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	log.Info("Открываем главную страницу для прохождения анти-бот проверки...")
	if err := chromedp.Run(chromedbCtx,
		chromedp.Navigate(baseURL),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		panic(err)
	}

	// Собираем полный список статей
	pages, err := p.getAllPages(chromedbCtx)
	if err != nil {
		panic(err)
	}
	log.Info("Найдены статьи", slog.Int("Количество", len(pages)))

	if len(pages) == 0 {
		log.Error("Список статей пуст")
		return fmt.Errorf("%s:Список статей пуст", op)
	}

	points := make([]*dto.Point, 0, p.batchSize)

	// 5. Идём по всем URL
	var id uint64
	id = 0
	var article dto.Article
	for i, page := range pages {
		title, content, err := p.scrapeArticle(chromedbCtx, page.URL)
		if err != nil {
			log.Error("Ошибка при парсинге", slog.Any("error", err), slog.String("url", page.URL))
			continue
		}
		if title == "" {
			title = page.Title
		}
		article = dto.Article{Title: title, Content: content, URL: page.URL}
		chunks := p.chunkArticle(article)
		for _, chunk := range chunks {
			id++
			req := &api.EmbedRequest{
				Model: "bge-m3",
				Input: chunk.Text,
			}
			resp, err := p.ollamaClient.Embed(ctx, req)
			if err != nil {
				log.Error("Не удалось получить эмбеддинг от bge-m3", slog.Any("error", err))
			}
			point := &dto.Point{
				Id:        id,
				Embedding: resp.Embeddings[0],
				Title:     fmt.Sprintf("%s: %s", chunk.SectionTitle, chunk.ArticleTitle),
				Content:   chunk.Text,
				URL:       chunk.SourceURL,
			}
			points = append(points, point)
			if len(points) >= p.batchSize {
				if err = p.qdrant.Upsert(ctx, points); err != nil {
					log.Error("Не удалось сохранить поинтеры в qdrant", slog.Any("error", err))
				} else {
					log.Info("Поинтеры успешно сохранены")
				}
				points = points[:0]
			}
			log.Info("Запаршен чанк", slog.Uint64("Номер", id))
		}
		log.Info("Запаршена статья", slog.Int("Номер", i+1), slog.String("заголовок", title))

		time.Sleep(500 * time.Millisecond)

	}
	if len(points) != 0 {
		if err = p.qdrant.Upsert(ctx, points); err != nil {
			log.Error("Не удалось сохранить поинтеры в qdrant", slog.Any("error", err))
		} else {
			log.Info("Финальные поинтеры успешно сохранены")
		}

	}
	return nil
}

func (p *Parser) getAllPages(ctx context.Context) ([]PageEntry, error) {
	var op = "service.parser.getAllPages"
	log := p.log.With(slog.String("op", op))
	var allPages []PageEntry
	seen := make(map[string]bool)

	currentURL := baseURL + "/index.php?title=Special:AllPages"

	for pageNum := 1; ; pageNum++ {
		log.Info("Special:AllPages", slog.Int("Страница", pageNum), slog.String("URL", currentURL))

		var links []map[string]string
		var nextURL string

		err := chromedp.Run(ctx,
			chromedp.Navigate(currentURL),
			// Ждём, пока в DOM появится хоть какая-то ссылка на статью —
			// надёжнее фиксированного Sleep, т.к. не зависит от скорости
			// прохождения анти-бот проверки на конкретном прогоне
			chromedp.WaitVisible(`body`, chromedp.ByQuery),
			chromedp.Sleep(3*time.Second),

			// Пробуем сразу несколько вариантов селекторов, объединяя
			// результат — это переживёт разные версии/скины MediaWiki:
			// - .mw-allpages-table-chunk a  (стандартный Vector)
			// - .mw-allpages-chunk a         (старые версии)
			// - #mw-content-text ul li a     (запасной вариант, если это
			//                                  просто список, а не таблица)
			chromedp.Evaluate(`
				(() => {
					const selectors = [
						'.mw-allpages-table-chunk a',
						'.mw-allpages-chunk a',
						'#mw-content-text table a',
						'#mw-content-text ul li a'
					];
					let found = [];
					for (const sel of selectors) {
						const els = document.querySelectorAll(sel);
						if (els.length > 0) {
							found = Array.from(els);
							break;
						}
					}
					return found
						.filter(a => !a.className.includes('new')) // пропускаем несуществующие статьи
						.map(a => ({title: a.textContent.trim(), href: a.href}));
				})()
			`, &links),

			chromedp.Evaluate(`
				(() => {
					const next = document.querySelector('a.mw-nextlink');
					return next ? next.href : '';
				})()
			`, &nextURL),
		)
		if err != nil {
			return nil, fmt.Errorf("%s: ошибка на странице %d (%s): %w", op, pageNum, currentURL, err)
		}

		added := 0
		for _, l := range links {
			href := l["href"]
			title := l["title"]
			if href == "" || title == "" || seen[href] {
				continue
			}
			seen[href] = true
			allPages = append(allPages, PageEntry{Title: title, URL: href})
			added++
		}
		slog.Info("добавлены новые страницы", slog.Int("Количество", added), slog.Int("Всего", len(allPages)))

		if nextURL == "" || nextURL == currentURL {
			slog.Info("Пагинация закончилась.")
			break
		}
		currentURL = nextURL
		time.Sleep(500 * time.Millisecond)
	}

	return allPages, nil
}

func (p *Parser) scrapeArticle(ctx context.Context, url string) (title, content string, err error) {
	var op = "service.parser.crapeArticle"
	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`#mw-content-text`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Text(`#firstHeading`, &title, chromedp.NodeVisible),
		chromedp.Text(`#mw-content-text`, &content, chromedp.NodeVisible),
	)
	if err != nil {
		return "", "", fmt.Errorf("%s:%w", op, err)
	}
	return strings.TrimSpace(title), strings.TrimSpace(content), nil
}
