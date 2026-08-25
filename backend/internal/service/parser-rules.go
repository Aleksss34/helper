package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Aleksss34/helper/backend/internal/dto"
	"github.com/chromedp/chromedp"
)

var (
	nbspReplacer      = strings.NewReplacer("\u00a0", " ")
	multiSpaceRegex   = regexp.MustCompile(`[ \t]+`)
	multiNewlineRegex = regexp.MustCompile(`\n{2,}`)
)

const baseURLRules = "https://amazing-online.com/rules/"

// Заполни реальными слагами страниц: 1-atmosfera, 2-obschenie и т.д.
var rulePages = []string{
	"1-atmosfera",
	"2-obschenie",
	"3-sityatsii",
	"4-terminologiya",
	"5-nik-igrovogo-personazha",
	"6-vdali-ot-klaviatyri",
	"7-dorozhnoe-dvizhenie",
	"8-akkaynt-i-imyschestvo",
	"9-pravila-organizatsiy",
	"10-pravila-oblav",
	"11-nechestnoe-preimyschestvo",
	"12-golosovoy-chat-i-aydiosistemi",
	"13-proniknovenie-na-obekti",
	"14-zadachi-organizatsiy",
	"15-pravila-prinyatiya-i-perevodov",
	"16-ystav-liderov-i-helperov",
	"17-kompanii-taksoparki-i-semi",
	"18-voyni-za-territorii-i-streli",
	"19-pravila-vedeniya-boya",
	"20-reyd-korablekryshenie-i-tseh",
	"21-sistemnie-ogrableniya-i-ygoni",
	"22-poezd-i-ograblenie-konteynerov",
	"23-semeynoe-ograblenie",
	"24-sistemnie-meropriyatiya",
	"25-pravila-strel-i-zahvatov",
	"26-pravila-pohischeniy",
	"27-pravila-sleta-nedvizhimosti",
}

const rulesContainerSelector = "main" // если пусто - используется body
const rulesHeadingSelector = "h1,h2,h3,h4,h5"

// Мягкий лимит на чанк из обычного текста (не таблицы). Если блок под
// заголовком больше — режем на части.
const maxRulesTextWords = 150

// RuleBlock — один блок страницы в порядке документа: либо таблица
// (уже развёрнутая построчно, с учётом rowspan), либо кусок обычного текста
// (абзацы/списки), который шёл под текущим заголовком до следующего
// заголовка/таблицы.
type RuleBlock struct {
	Type    string // "table" | "text"
	Label   string // текст заголовка над блоком, например "Запрещено"
	Headers []string
	Rows    []map[string]string
	Text    string
}

func (p *Parser) ParseRules(ctx context.Context) error {
	var op = "service.parser.ParseRules"
	log := p.log.With(slog.String("op", op))

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(p.browserPath),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	chromedbCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	points := make([]*dto.Point, 0, p.batchSize)
	var id uint64 = 0

	for _, slug := range rulePages {
		pageURL := baseURLRules + slug
		title, blocks, err := p.scrapeRulesPage(chromedbCtx, pageURL)
		if err != nil {
			log.Error("не удалось запарсить страницу правил", slog.String("URL", pageURL), slog.Any("error", err))
			continue
		}
		log.Info("Получена страница правил", slog.String("slug", slug), slog.String("title", title), slog.Int("блоков", len(blocks)))

		chunks := p.chunkRulesBlocks(title, pageURL, blocks)
		for _, chunk := range chunks {
			id++
			pointId := p.hashToUint64(chunk.SourceURL + "#" + strconv.Itoa(int(id)))
			point := p.getPoint(ctx, chunk, pointId, p.vocab, p.avgDL)
			points = append(points, point)
			if len(points) >= p.batchSize {
				if err = p.qdrant.Upsert(ctx, points); err != nil {
					log.Error("Не удалось сохранить поинтеры в qdrant", slog.Any("error", err))
				} else {
					log.Info("Поинтеры успешно сохранены")
					if err := p.vocab.Save(); err != nil {
						log.Error("не удалось сохранить bm25-словарь", slog.Any("error", err))
					}
				}
				points = points[:0]
			}
			log.Info("Запаршен чанк правил", slog.Uint64("Номер", id))
		}

		time.Sleep(500 * time.Millisecond)
	}

	if len(points) != 0 {
		if err := p.qdrant.Upsert(ctx, points); err != nil {
			log.Error("Не удалось сохранить поинтеры в qdrant", slog.Any("error", err))
		} else {
			log.Info("Финальные поинтеры успешно сохранены")
			if err := p.vocab.Save(); err != nil {
				log.Error("не удалось сохранить bm25-словарь", slog.Any("error", err))
			}
		}
	}

	return nil
}

// scrapeRulesPage извлекает данные с учётом <p><b> (заголовки) и <table> (строки таблиц)
func (p *Parser) scrapeRulesPage(ctx context.Context, url string) (title string, blocks []RuleBlock, err error) {
	var op = "service.parser-rules.scrapeRulesPage"

	var raw []struct {
		Type    string              `json:"type"`
		Label   string              `json:"label"`
		Headers []string            `json:"headers"`
		Rows    []map[string]string `json:"rows"`
		Text    string              `json:"text"`
	}

	err = chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.Title(&title),
		chromedp.Evaluate(`
			(() => {
				function parseTable(table) {
					const rows = Array.from(table.querySelectorAll('tr'));
					if (rows.length === 0) return {headers: [], rows: []};

					// Находим заголовки колонок (Описание, Наказание, Причина и т.д.)
					const headerCells = Array.from(rows[0].querySelectorAll('th,td'));
					const headers = headerCells.map(c => c.innerText.trim());

					const carry = new Array(headers.length).fill(null);
					const result = [];

					for (let r = 1; r < rows.length; r++) {
						const cells = Array.from(rows[r].querySelectorAll('td,th'));
						const rowValues = new Array(headers.length).fill('');
						let cellIdx = 0;

						for (let col = 0; col < headers.length; col++) {
							if (carry[col] && carry[col].remaining > 0) {
								rowValues[col] = carry[col].value;
								carry[col].remaining--;
								continue;
							}
							const cell = cells[cellIdx];
							if (!cell) continue;

							// Клонируем ячейку, чтобы заменять <br> на переносы
							const clone = cell.cloneNode(true);
							clone.querySelectorAll('br').forEach(br => br.replaceWith(' '));
							let text = clone.innerText.replace(/\s+/g, ' ').trim();
							

							const rowspan = parseInt(cell.getAttribute('rowspan') || '1', 10);
							rowValues[col] = text;
							if (rowspan > 1) {
								carry[col] = {value: text, remaining: rowspan - 1};
							}
							cellIdx++;
						}

						const nonEmpty = rowValues.some(v => v.length > 0);
						if (!nonEmpty) continue;

						const obj = {};
						headers.forEach((h, i) => { obj[h] = rowValues[i] || ''; });
						result.push(obj);
					}

					return {headers, rows: result};
				}

				const container = document.querySelector("main") || document.body;
				// Селектируем абсолютно ВСЕ нужные узлы (включая таблицы)
				const nodes = container.querySelectorAll('p, center, blockquote, table');
				
				let currentLabel = '';
				const blocks = [];

				nodes.forEach(node => {
					// 1. Ищем заголовок типа <p><b>Запрещено</b></p>
					const bEl = node.querySelector('b');
					if ((node.tagName === 'P' || node.tagName === 'CENTER') && bEl) {
						const labelText = bEl.innerText.trim();
						if (labelText) {
							currentLabel = labelText;
						}
						return;
					}

					// 2. Если встретили ТАБЛИЦУ (она может быть как внутри blockquote, так и сама по себе)
					if (node.tagName === 'TABLE') {
						const parsed = parseTable(node);
						if (parsed.rows.length > 0) {
							blocks.push({
								type: 'table',
								label: currentLabel,
								headers: parsed.headers,
								rows: parsed.rows
							});
						}
						return;
					}

					// 3. Если это обычный текст blockquote (без таблиц внутри)
					if (node.tagName === 'BLOCKQUOTE' && !node.querySelector('table')) {
						const clone = node.cloneNode(true);
						clone.querySelectorAll('br').forEach(br => br.replaceWith('\n'));
						const text = clone.innerText.trim();
						if (text) {
							blocks.push({
								type: 'text',
								label: currentLabel,
								text: text
							});
						}
					}
				});

				return blocks;
			})()
		`, &raw),
	)
	if err != nil {
		return "", nil, fmt.Errorf("%s:%w", op, err)
	}

	for _, b := range raw {
		blocks = append(blocks, RuleBlock{
			Type:    b.Type,
			Label:   strings.TrimSpace(b.Label),
			Headers: b.Headers,
			Rows:    b.Rows,
			Text:    b.Text,
		})
	}
	if idx := strings.Index(title, "—"); idx != -1 {
		title = title[:idx]
	}
	return strings.TrimSpace(title), blocks, nil
}
