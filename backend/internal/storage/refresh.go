package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func (r *Redis) AddRefresh(ctx context.Context, tokenHash string, userId int64) error {
	var op = "storage.refresh.Add"
	cmd := r.rdb.Set(ctx, "refresh:"+tokenHash, userId, time.Duration(r.timeoutToken)*time.Hour)
	if cmd.Err() != nil {
		return fmt.Errorf("%s:%w", op, cmd.Err())
	}
	return nil
}

func (r *Redis) GetIdByRefresh(ctx context.Context, tokenHash string) (string, error) {
	var op = "storage.refresh.GetId"
	userId, err := r.rdb.Get(ctx, "refresh:"+tokenHash).Result()
	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("%s:invalid refresh token", op)
	}
	return userId, nil
}

func (r *Redis) DelRefresh(ctx context.Context, tokenHash string) error {
	var op = "storage.refresh.Del"
	cmd := r.rdb.Del(ctx, tokenHash)
	if cmd.Err() != nil {
		return fmt.Errorf("%s:%w", op, cmd.Err())
	}
	return nil
}
