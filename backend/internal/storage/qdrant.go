package storage

import (
	"context"
	"fmt"
	"gateway/backend/internal/dto"

	"github.com/qdrant/go-client/qdrant"
)

func (q *Qdrant) Upsert(ctx context.Context, points []*dto.Point) error {
	var op = "storage.qdrant.Upsert"
	qdrantPoints := make([]*qdrant.PointStruct, 0, len(points))
	for _, p := range points {
		qdrantPoint := &qdrant.PointStruct{Id: qdrant.NewIDNum(p.Id),
			Vectors: qdrant.NewVectors(p.Embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"title":   p.Title,
				"content": p.Content,
			})}
		qdrantPoints = append(qdrantPoints, qdrantPoint)
	}
	if _, err := q.db.Upsert(ctx, &qdrant.UpsertPoints{CollectionName: q.collectionName, Points: qdrantPoints}); err != nil {
		return fmt.Errorf("%s:%w", op, err)
	}
	return nil
}

func (q *Qdrant) Get(ctx context.Context, embedding []float32) ([]*dto.Point, error) {
	var op = "storage.qdrant.Get"
	var resp []*dto.Point
	var point *dto.Point

	scoredPoints, err := q.db.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collectionName,
		Query:          qdrant.NewQuery(embedding...),
		Limit:          &q.limitPoints,
		ScoreThreshold: &q.scoreThreshold,
		WithPayload:    qdrant.NewWithPayload(true)})
	if err != nil {
		return nil, fmt.Errorf("%s:%w", op, err)
	}

	for _, p := range scoredPoints {

		url := p.Payload["URL"].GetStringValue()
		title := p.Payload["title"].GetStringValue()
		content := p.Payload["content"].GetStringValue()
		point = &dto.Point{Id: p.Id.GetNum(), URL: url, Title: title, Content: content}
		resp = append(resp, point)
	}

	return resp, nil
}
