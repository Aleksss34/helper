package storage

import (
	"database/sql"

	"github.com/qdrant/go-client/qdrant"
	"github.com/redis/go-redis/v9"
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
type Redis struct {
	rdb          *redis.Client
	timeoutToken int64
}
type Storage struct {
	qdrant   *Qdrant
	postgres *Postgres
	redis    *Redis
}

func NewQdrant(collectionName string, db *qdrant.Client, limitPoints uint64, scoreThreshold float32) *Qdrant {
	return &Qdrant{collectionName: collectionName, db: db, limitPoints: limitPoints, scoreThreshold: scoreThreshold}
}
func NewPostgres(db *sql.DB) *Postgres {
	return &Postgres{db: db}
}
func NewRedis(rdb *redis.Client, timeoutToken int64) *Redis {
	return &Redis{
		rdb:          rdb,
		timeoutToken: timeoutToken,
	}
}
func NewStorage(qdrant *Qdrant, postgres *Postgres, redis *Redis) *Storage {
	return &Storage{qdrant: qdrant, postgres: postgres, redis: redis}
}
