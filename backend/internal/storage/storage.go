package storage

import (
	"database/sql"

	"github.com/qdrant/go-client/qdrant"
)

type Qdrant struct {
	collectionName string
	limitPoints    uint64
	scoreThreshold float32
	db             *qdrant.Client
}

type Postgres struct {
	db *sql.DB
}
type Storage struct {
}

func NewQdrant(collectionName string, db *qdrant.Client, limitPoints uint64, scoreThreshold float32) *Qdrant {
	return &Qdrant{collectionName: collectionName, db: db, limitPoints: limitPoints, scoreThreshold: scoreThreshold}
}
func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}
