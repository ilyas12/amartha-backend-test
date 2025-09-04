package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func OpenRedis(addr string, db int) (*redis.Client, error) {
	r := redis.NewClient(&redis.Options{Addr: addr, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return r, nil
}
