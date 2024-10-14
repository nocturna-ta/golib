package redis

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"golib/cache"
	"golib/tracing"
	"net/url"
	"strings"
	"time"
)

const (
	defaultNS = "redis"
)

type Cache struct {
	client *redis.Client
	ns     string
}

func init() {
	cache.Register("redis", NewCache)
}

func NewCache(u *url.URL) (cache.Cache, error) {
	p, _ := u.User.Password()
	opt := &redis.Options{
		Addr:     u.Host,
		Password: p,
		DB:       0, // use default DB
	}

	if ts := u.Query().Get("tls"); ts != "" {
		opt.TLSConfig = &tls.Config{
			ServerName: ts,
		}
	}

	rClient := redis.NewClient(opt)

	ns := strings.TrimPrefix(u.Path, "/")
	if ns == "" {
		ns = defaultNS
	}

	redisCache := &Cache{
		client: rClient,
		ns:     ns,
	}
	_, err := redisCache.client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return redisCache, nil
}

func (c *Cache) Set(ctx context.Context, key string, value any, expiration int) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.Set")
	defer span.End()

	switch value.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, []byte:
		return c.client.Set(ctx, c.ns+key, value, time.Duration(expiration)*time.Second).Err()
	default:
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return c.client.Set(ctx, c.ns+key, b, time.Duration(expiration)*time.Second).Err()
	}
}

func (c *Cache) Increment(ctx context.Context, key string, expiration int) (int64, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.Increment")
	defer span.End()

	switch expiration {
	case 0:
		i, err := c.client.Incr(ctx, key).Result()
		if err != nil {
			return 0, err
		}
		return i, nil
	default:
		pipe := c.client.TxPipeline()

		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Second*time.Duration(expiration))

		_, err := pipe.Exec(ctx)
		if err != nil {
			return 0, err
		}
		return incr.Val(), nil
	}
}

func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.Get")
	defer span.End()

	b, err := c.client.Get(ctx, c.ns+key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, cache.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func (c *Cache) GetObject(ctx context.Context, key string, doc any) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.GetObject")
	defer span.End()

	b, err := c.client.Get(ctx, c.ns+key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return cache.ErrNotFound
		}
		return err
	}
	return json.Unmarshal(b, doc)
}

func (c *Cache) GetString(ctx context.Context, key string) (string, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.GetString")
	defer span.End()

	s, err := c.client.Get(ctx, c.ns+key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", cache.ErrNotFound
		}
		return "", err
	}
	return s, nil
}

func (c *Cache) GetInt(ctx context.Context, key string) (int64, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.GetInt")
	defer span.End()

	i, err := c.client.Get(ctx, c.ns+key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, cache.ErrNotFound
		}
		return 0, err
	}
	return i, nil
}

func (c *Cache) GetFloat(ctx context.Context, key string) (float64, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.GetFloat")
	defer span.End()

	f, err := c.client.Get(ctx, c.ns+key).Float64()
	if err != nil {
		if err == redis.Nil {
			return 0, cache.ErrNotFound
		}
		return 0, err
	}
	return f, nil
}

func (c *Cache) Exist(ctx context.Context, key string) bool {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.Exist")
	defer span.End()

	return c.client.Exists(ctx, c.ns+key).Val() > 0
}

func (c *Cache) Delete(ctx context.Context, key string, opts ...cache.DeleteOptions) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.Delete")
	defer span.End()

	deleteCache := &cache.DeleteCache{}
	for _, opt := range opts {
		opt(deleteCache)
	}

	if deleteCache.Pattern != "" {
		return c.deletePattern(ctx, deleteCache.Pattern)
	}
	return c.client.Del(ctx, c.ns+key).Err()
}

func (c *Cache) GetKeys(ctx context.Context, pattern string) []string {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.GetKeys")
	defer span.End()

	cmd := c.client.Keys(ctx, pattern)
	keys, err := cmd.Result()
	if err != nil {
		return nil
	}
	return keys
}

func (c *Cache) RemainingTime(ctx context.Context, key string) int {
	span, ctx := tracing.StartSpanFromContext(ctx, "Redis.RemainingTime")
	defer span.End()

	return int(c.client.TTL(ctx, c.ns+key).Val().Seconds())
}

func (c *Cache) Close() error {
	return c.client.Close()
}

func (c *Cache) deletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, c.ns+pattern, 0).Iterator()
	var localKeys []string

	for iter.Next(ctx) {
		localKeys = append(localKeys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(localKeys) > 0 {
		_, err := c.client.Pipelined(ctx, func(pipeline redis.Pipeliner) error {
			pipeline.Del(ctx, localKeys...)
			return nil
		})

		if err != nil {
			return err
		}
	}

	return nil
}
