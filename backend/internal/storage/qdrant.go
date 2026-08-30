package storage

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Aleksss34/helper/backend/internal/domain"
	"github.com/qdrant/go-client/qdrant"
)

func (q *Qdrant) Upsert(ctx context.Context, points []*domain.Point) error {
	var op = "storage.qdrant.Upsert"
	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))

	for _, p := range points {
		vectors := map[string]*qdrant.Vector{
			"dense": qdrant.NewVector(p.Dense...),
		}

		if len(p.SparseIdx) > 0 {
			vectors["sparse"] = qdrant.NewVectorSparse(p.SparseIdx, p.SparseVal)
		}

		qdrantPoint := &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(p.Id),
			Vectors: qdrant.NewVectorsMap(vectors),
			Payload: qdrant.NewValueMap(map[string]any{
				"title":          p.Title,
				"content":        p.Content,
				"server":         p.Server,
				"URL":            p.URL,
				"article_number": p.ArticleNumber,
				"article_title":  p.ArticleTitle,
				"chapter_number": p.ChapterNumber,
			}),
		}
		qdrantPoints = append(qdrantPoints, qdrantPoint)
	}

	_, err := q.db.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points:         qdrantPoints,
		Wait:           qdrant.PtrOf(true),
	})
	if err != nil {
		return fmt.Errorf("%s:%w", op, err)
	}

	return nil
}

func (q *Qdrant) EnsureExactFilterIndexes(ctx context.Context) error {
	var op = "storage.qdrant.EnsureExactFilterIndexes"

	fields := []string{"article_number", "article_title"}
	for _, field := range fields {
		_, err := q.db.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: q.collectionName,
			FieldName:      field,
			FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		})
		if err != nil {
			return fmt.Errorf("%s: индекс на поле %s: %w", op, field, err)
		}
	}
	return nil
}
func (q *Qdrant) ExactSearch(ctx context.Context, articleNumber, articleTitle, server string) ([]*domain.Point, error) {
	var op = "storage.qdrant.ExactSearch"

	if articleNumber == "" {
		return nil, fmt.Errorf("%s: articleNumber не может быть пустым", op)
	}

	must := []*qdrant.Condition{
		qdrant.NewMatch("article_number", articleNumber),
	}
	if articleTitle != "" {
		must = append(must, qdrant.NewMatch("article_title", articleTitle))
	}
	if server != "" {
		must = append(must, qdrant.NewMatchKeywords("server", server, "all"))
	}

	result, err := q.db.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: q.collectionName,
		Filter:         &qdrant.Filter{Must: must},
		Limit:          qdrant.PtrOf(uint32(10)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	var points []*domain.Point
	for _, p := range result {
		points = append(points, &domain.Point{
			Id:            p.Id.GetNum(),
			Title:         p.Payload["title"].GetStringValue(),
			Content:       p.Payload["content"].GetStringValue(),
			URL:           p.Payload["URL"].GetStringValue(),
			Server:        p.Payload["server"].GetStringValue(),
			ArticleNumber: p.Payload["article_number"].GetStringValue(),
			ArticleTitle:  p.Payload["article_title"].GetStringValue(),
		})
	}
	return points, nil
}

// SearchByChapter возвращает ВСЕ точки (статьи), относящиеся к указанной
// главе указанного закона, отсортированные по номеру статьи по
// возрастанию — порядок важен, т.к. далее эти чанки уходят в LLM как
// единый последовательный текст главы.
func (q *Qdrant) SearchByChapter(
	ctx context.Context,
	chapterNumber string,
	lawName string,
	server string,
) ([]*domain.Point, error) {

	must := []*qdrant.Condition{
		qdrant.NewMatch("chapter_number", chapterNumber),
	}

	if lawName != "" {
		must = append(must, qdrant.NewMatch("article_title", lawName))
	}
	if server != "" {
		must = append(must, qdrant.NewMatchKeywords("server", server, "all"))
	}
	filter := &qdrant.Filter{
		Must: must,
	}

	res, err := q.db.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: q.collectionName,
		Filter:         filter,
		Limit:          qdrant.PtrOf(uint32(20)),
		WithPayload:    qdrant.NewWithPayload(true),
		WithVectors:    qdrant.NewWithVectorsEnable(false),
	})
	if err != nil {
		return nil, fmt.Errorf("scroll по главе %s: %w", chapterNumber, err)
	}

	points := make([]*domain.Point, 0, len(res))
	for _, p := range res {
		points = append(points, &domain.Point{
			Id:            p.Id.GetNum(),
			Title:         p.Payload["title"].GetStringValue(),
			Content:       p.Payload["content"].GetStringValue(),
			URL:           p.Payload["URL"].GetStringValue(),
			Server:        p.Payload["server"].GetStringValue(),
			ArticleNumber: p.Payload["article_number"].GetStringValue(),
			ArticleTitle:  p.Payload["article_title"].GetStringValue(),
		})

	}

	sort.Slice(points, func(i, j int) bool {
		return articleNumberLess(points[i].ArticleNumber, points[j].ArticleNumber)
	})

	return points, nil
}

// articleNumberLess сравнивает номера статей вида "9", "9.5", "13.11"
// как числа, а не строки, чтобы "10" не оказалось перед "2" и "9.2"
// шло раньше "9.10".
func articleNumberLess(a, b string) bool {
	aParts := strings.SplitN(a, ".", 2)
	bParts := strings.SplitN(b, ".", 2)

	aMain, _ := strconv.Atoi(aParts[0])
	bMain, _ := strconv.Atoi(bParts[0])

	if aMain != bMain {
		return aMain < bMain
	}

	var aSub, bSub int
	if len(aParts) > 1 {
		aSub, _ = strconv.Atoi(aParts[1])
	}
	if len(bParts) > 1 {
		bSub, _ = strconv.Atoi(bParts[1])
	}

	return aSub < bSub
}
func (q *Qdrant) Get(ctx context.Context, dense, sparseVal []float32, sparseIdx []uint32, server string) ([]*domain.Point, error) {

	var op = "storage.qdrant.Get"
	var resp []*domain.Point
	var point *domain.Point

	scoredPoints, err := q.db.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collectionName,
		Prefetch: []*qdrant.PrefetchQuery{
			{
				Query: qdrant.NewQueryDense(dense),
				Using: qdrant.PtrOf("dense"),
				Limit: qdrant.PtrOf(uint64(50)),
			},
			{
				Query: qdrant.NewQuerySparse(sparseIdx, sparseVal),
				Using: qdrant.PtrOf("sparse"),
				Limit: qdrant.PtrOf(uint64(50)),
			},
		},
		Query: qdrant.NewQueryFusion(qdrant.Fusion_RRF),
		Limit: &q.limitPoints,

		WithPayload: qdrant.NewWithPayload(true),
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{
				qdrant.NewMatchKeywords("server", server, "all"),
			},
		}})
	if err != nil {
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	for _, p := range scoredPoints {

		url := p.Payload["URL"].GetStringValue()
		title := p.Payload["title"].GetStringValue()
		content := p.Payload["content"].GetStringValue()
		serv := p.Payload["server"].GetStringValue()
		point = &domain.Point{Id: p.Id.GetNum(), URL: url, Title: title, Content: content, Server: serv}
		resp = append(resp, point)
	}

	return resp, nil
}

// тестовый
//func (q *Qdrant) Get(
//	ctx context.Context,
//	dense []float32,
//	sparseVal []float32,
//	sparseIdx []uint32,
//	server string,
//) ([]*domain.Point, error) {
//
//	filter := &qdrant.Filter{
//		Must: []*qdrant.Condition{
//			qdrant.NewMatchKeywords("server", server, "all"),
//		},
//	}
//
//	// ===== DENSE =====
//	denseResult, err := q.db.Query(ctx, &qdrant.QueryPoints{
//		CollectionName: q.collectionName,
//		Query:          qdrant.NewQueryDense(dense),
//		Using:          qdrant.PtrOf("dense"),
//		Limit:          qdrant.PtrOf(uint64(20)),
//		WithPayload:    qdrant.NewWithPayload(true),
//		Filter:         filter,
//	})
//	if err != nil {
//		return nil, fmt.Errorf("dense search: %w", err)
//	}
//
//	fmt.Println("\n========== DENSE TOP 20 ==========")
//
//	for i, p := range denseResult {
//		fmt.Printf(
//			"%2d. score=%.6f id=%d title=%q\n",
//			i+1,
//			p.Score,
//			p.Id.GetNum(),
//			p.Payload["title"].GetStringValue(),
//		)
//	}
//
//	// ===== SPARSE =====
//	sparseResult, err := q.db.Query(ctx, &qdrant.QueryPoints{
//		CollectionName: q.collectionName,
//		Query:          qdrant.NewQuerySparse(sparseIdx, sparseVal),
//		Using:          qdrant.PtrOf("sparse"),
//		Limit:          qdrant.PtrOf(uint64(20)),
//		WithPayload:    qdrant.NewWithPayload(true),
//		Filter:         filter,
//	})
//	if err != nil {
//		return nil, fmt.Errorf("sparse search: %w", err)
//	}
//
//	fmt.Println("\n========== SPARSE TOP 20 ==========")
//
//	for i, p := range sparseResult {
//		fmt.Printf(
//			"%2d. score=%.6f id=%d title=%q\n",
//			i+1,
//			p.Score,
//			p.Id.GetNum(),
//			p.Payload["title"].GetStringValue(),
//		)
//	}
//
//	return nil, nil
//}

func (q *Qdrant) SearchSubArticles(
	ctx context.Context,
	articleNumber string,
	lawName string,
	server string,
) ([]*domain.Point, error) {

	var op = "storage.qdrant.SearchSubArticles"

	must := []*qdrant.Condition{
		qdrant.NewMatchKeywords("article_title", lawName, "all"),
		qdrant.NewMatchKeywords("server", server, "all"),
	}

	var result []*domain.Point

	// Для запроса статьи "9" ищем "9.1", "9.2", "9.3"...
	prefix := articleNumber + "."

	points, err := q.db.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: q.collectionName,
		Filter: &qdrant.Filter{
			Must: must,
		},
		WithPayload: qdrant.NewWithPayload(true),
		Limit:       qdrant.PtrOf(uint32(100)),
	})

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	for _, p := range points {
		if p.Payload == nil {
			continue
		}

		foundArticleNumber := p.Payload["article_number"].GetStringValue()

		if !strings.HasPrefix(foundArticleNumber, prefix) {
			continue
		}

		result = append(result, &domain.Point{
			Id:            p.Id.GetNum(),
			URL:           p.Payload["URL"].GetStringValue(),
			Title:         p.Payload["title"].GetStringValue(),
			Content:       p.Payload["content"].GetStringValue(),
			Server:        p.Payload["server"].GetStringValue(),
			ArticleNumber: foundArticleNumber,
			ArticleTitle:  p.Payload["article_title"].GetStringValue(),
		})
	}

	return result, nil
}
