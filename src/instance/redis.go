package instance

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Redis interface {
	Ping(ctx context.Context) error
	Subscribe(ctx context.Context, ch chan string, subscribeTo ...string)
	Get(ctx context.Context, key string) (interface{}, error)
	SetEX(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
	Pipeline(ctx context.Context) redis.Pipeliner
}
