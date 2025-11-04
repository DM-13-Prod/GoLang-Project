// internal/audit/redislogger.go
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"todo/internal/service"
)

type RedisLogger struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewRedisLogger(addr, password string, db int, ttl time.Duration, prefix string) *RedisLogger {
	return &RedisLogger{
		client: redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		}),
		ttl:    ttl,
		prefix: prefix,
	}
}

func (l *RedisLogger) LogEvent(ctx context.Context, e service.Event) error {
	raw, _ := json.Marshal(e)
	key := fmt.Sprintf("%s:%s:%d:%d", l.prefix, e.Op, e.TaskID, e.At.UnixNano())
	return l.client.Set(ctx, key, raw, l.ttl).Err()
}