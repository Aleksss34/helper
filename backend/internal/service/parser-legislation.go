package service

import (
	"context"
	"fmt"
	"log/slog"

	"strings"
	"time"

	"github.com/Aleksss34/helper/backend/internal/dto"
	"github.com/chromedp/chromedp"
)

const baseURLForum = "https://forum.amazing-online.com"
const forumSectionUrl = baseURLForum + "/forums/zakonodatelstvo."

var idPages = []string{"619", "620", "621", "519", "588", "622", "647", "732", "817", "875", "985", "1077"}
var serverNames = map[int]string{0: "RED", 1: "YELLOW", 2: "GREEN", 3: "AZURE", 4: "SILVER", 5: "ROSE", 6: "BLACK", 7: "SKY", 8: "TITAN", 9: "X", 10: "FIRE", 11: "LIME"}

type ThreadEntry struct {
	Title string
	URL   string
}

func (p *Parser) ParseLegislation(ctx context.Context) error {
	var op = "service.parser.ParseLegislation"
	log := p.log.With(slog.String("op", op))
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(p.browserPath),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	chromedbCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	log.Info("Открываем главную страницу форума для прохождения анти-бот проверки...")
	if err := chromedp.Run(chromedbCtx,
		chromedp.Navigate(baseURLForum),
		chromedp.Sleep(3*time.Second),
	); err != nil {
		panic("Не удалось открыть главную страницу форума, ошибка: " + err.Error())
	}

	points := make([]*dto.Point, 0, p.batchSize)
	var id uint64
	id = 0
	for i, idPage := range idPages {
		serverName := serverNames[i]
		currentURL := forumSectionUrl + idPage + "/"
		threads, err := p.getAllThreads(chromedbCtx, currentURL)
		if err != nil {
			panic("Не удалось получить треды, ошибка: " + err.Error())
		}
		for j, t := range threads {
			title, contents, err := p.scrapePosts(chromedbCtx, t.URL)
			if err != nil {
				log.Error("не удалось запарсить тред", slog.String("сервер", serverName), slog.Any("error", err), slog.String("тред", fmt.Sprintf("%d/%d", j+1, len(threads))), slog.String("URL", t.URL))
				continue
			}
			if title == "" {
				title = t.Title
			}
			title = strings.ReplaceAll(title, "ЗАКРЕПЛЕНО", "")
			title = strings.ReplaceAll(title, "ИНФОРМАЦИЯ", "")

			title = strings.TrimSpace(title)
			//////
			for _, content := range contents {
				article := dto.Article{Title: title, Content: content, URL: t.URL, Server: serverName}
				chunks := p.chunkLegislationArticle(article)

				for _, chunk := range chunks {
					id++
					point := p.getPoint(ctx, chunk, id)
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
		}
	}

	if len(points) != 0 {
		if err := p.qdrant.Upsert(ctx, points); err != nil {
			log.Error("Не удалось сохранить поинтеры в qdrant", slog.Any("error", err))
		} else {
			log.Info("Финальные поинтеры успешно сохранены")
		}

	}
	return nil
}
func (p *Parser) getAllThreads(ctx context.Context, currentURL string) ([]ThreadEntry, error) {
	var op = "service.parse-legislation.getAllThreads"
	log := p.log.With(slog.String("op", op))
	var allThreads []ThreadEntry
	seen := make(map[string]bool)

	for pageNum := 1; ; pageNum++ {
		log.Info("Получен раздел форума", slog.Int("Номер страницы", pageNum), slog.String("URL", currentURL))

		var threads []map[string]string
		var nextURL string

		err := chromedp.Run(ctx,
			chromedp.Navigate(currentURL),
			chromedp.WaitVisible(`body`, chromedp.ByQuery),
			chromedp.Sleep(3*time.Second),

			// Несколько вариантов селекторов под разные версии/темы XenForo:
			// - .structItem-title a          (актуальный XenForo 2.x)
			// - .structItemContainer a.structItem-title
			// - a[data-tp-primary="on"]      (запасной, ссылка-заголовок темы)
			chromedp.Evaluate(`
				(() => {
					const selectors = [
						'.structItem-title a[href*="/threads/"]',
						'.structItemContainer a.structItem-title',
						'a[data-tp-primary="on"]'
					];
					let found = [];
					for (const sel of selectors) {
						const els = document.querySelectorAll(sel);
						if (els.length > 0) {
							found = Array.from(els);
							break;
						}
					}
					// Дедуп внутри страницы + убираем служебные ссылки (например,
					// на страницы внутри той же темы вида thread-name.123/page-2)
					const seenHref = new Set();
					const result = [];
					for (const a of found) {
						const href = a.href.split('#')[0];
						if (seenHref.has(href)) continue;
						seenHref.add(href);
						result.push({title: a.textContent.trim(), href});
					}
					return result;
				})()
			`, &threads),

			// Пагинация: ищем ссылку "Дальше"/"Next" внизу списка тем
			chromedp.Evaluate(`
				(() => {
					const next = document.querySelector('.pageNav-jump--next, a.pageNav-jump--next');
					return next ? next.href : '';
				})()
			`, &nextURL),
		)
		if err != nil {
			return nil, fmt.Errorf("%s:ошибка на странице %d (%s): %w", op, pageNum, currentURL, err)
		}

		added := 0
		for _, t := range threads {
			href := t["href"]
			title := t["title"]
			if href == "" || title == "" || seen[href] {
				continue
			}
			seen[href] = true
			allThreads = append(allThreads, ThreadEntry{Title: title, URL: href})
			added++
		}
		log.Info("Добавлены новые темы", slog.Int("Количество", added), slog.Int("Всего", len(allThreads)))
		if nextURL == "" || nextURL == currentURL {
			log.Info("Пагинация закончилась")
			break
		}
		currentURL = nextURL
		time.Sleep(500 * time.Millisecond)
	}

	return allThreads, nil
}

func (P *Parser) scrapePosts(ctx context.Context, url string) (title string, contents []string, err error) {
	var op = "service.parse-legislation.scrapePosts"
	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`.message-body`, chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		// Заголовок темы
		chromedp.Text(`.p-title-value`, &title, chromedp.NodeVisible),

		// Получаем все сообщения
		chromedp.Evaluate(`
			(() => {
				const els = document.querySelectorAll('.message--post .message-body .bbWrapper');
				return Array.from(els).map(el => el.innerText.trim()).filter(text => text.length > 0);
			})()
		`, &contents),
	)
	if err != nil {
		return "", nil, fmt.Errorf("%s:%w", op, err)
	}
	for i := 0; i < len(contents); i++ {
		contents[i] = strings.TrimSpace(contents[i])
	}
	return strings.TrimSpace(title), contents, nil
}
