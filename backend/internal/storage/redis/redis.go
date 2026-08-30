package redis

import (
	"context"
	"fmt"
	"os"

	"github.com/Aleksss34/helper/backend/internal/domain"
	"github.com/redis/go-redis/v9"
)

func Conn(ctx context.Context, redisParams domain.RedisParams) (*redis.Client, error) {
	var op = "storage.redis.Conn"
	fmt.Println(redisParams.Password)
	fmt.Println(os.Getenv("REDIS_PASS"))
	addr := fmt.Sprintf("%s:%s", redisParams.Host, redisParams.Port)
	rdb := redis.NewClient(&redis.Options{Addr: addr, Password: redisParams.Password})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s:%w", op, err)
	}
	return rdb, nil
}
