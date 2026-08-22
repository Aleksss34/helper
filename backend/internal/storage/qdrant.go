package storage

import (
	"context"
	"fmt"

	"github.com/Aleksss34/helper/backend/internal/dto"

	"github.com/qdrant/go-client/qdrant"
)

func (q *Qdrant) Upsert(ctx context.Context, points []*dto.Point) error {
	var op = "storage.qdrant.Upsert"
	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))

	for _, p := range points {
		vectors := map[string]*qdrant.Vector{
			"dense": qdrant.NewVector(p.Dense...),
		}
		// Sparse-вектор добавляем, только если он реально есть -
		// точка без "sparse" ключа просто не будет участвовать в
		// sparse-поиске, но это не ошибка (например, если для чанка
		// не нашлось ни одного значащего токена после Tokenize).
		if len(p.SparseIdx) > 0 {
			vectors["sparse"] = qdrant.NewVectorSparse(p.SparseIdx, p.SparseVal)
		}

		qdrantPoint := &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(p.Id),
			Vectors: qdrant.NewVectorsMap(vectors),
			Payload: qdrant.NewValueMap(map[string]any{
				"title":   p.Title,
				"content": p.Content,
				"server":  p.Server,
				"URL":     p.URL,
			}),
		}
		qdrantPoints = append(qdrantPoints, qdrantPoint)
	}

	res, err := q.db.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points:         qdrantPoints,
		Wait:           qdrant.PtrOf(true),
	})
	if err != nil {
		return fmt.Errorf("%s:%w", op, err)
	}
	fmt.Println(res.GetStatus())
	info, err := q.db.GetCollectionInfo(ctx, q.collectionName)
	if err == nil {
		fmt.Println(info.GetPointsCount())
	}
	return nil
}

func (q *Qdrant) Get(ctx context.Context, dense, sparseVal []float32, sparseIdx []uint32, server string) ([]*dto.Point, error) {

	var op = "storage.qdrant.Get"
	var resp []*dto.Point
	var point *dto.Point

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
		point = &dto.Point{Id: p.Id.GetNum(), URL: url, Title: title, Content: content, Server: serv}
		resp = append(resp, point)
	}

	return resp, nil
}

//тестовый
//func (q *Qdrant) Get(
//	ctx context.Context,
//	dense []float32,
//	sparseVal []float32,
//	sparseIdx []uint32,
//	server string,
//) ([]*dto.Point, error) {
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
