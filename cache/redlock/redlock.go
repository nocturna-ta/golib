package redlock

import (
	"context"
	goRedisLib "github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"github.com/nocturna-ta/golib/custerr"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/response"
	"github.com/nocturna-ta/golib/tracing"
	"github.com/nocturna-ta/golib/utils/syncmap"

	"net/url"
	"strings"
	"time"
)

var (
	defaultNs = "redlock:"
)

type RedLock interface {
	AcquireLock(ctx context.Context, key string, attempts int) error
	AcquireLockWithTTL(ctx context.Context, key string, ttl time.Duration, attempts int) error
	ReleaseLock(ctx context.Context, key string) error
}

type redLock struct {
	redSync     *redsync.Redsync
	ns          string
	currentLock *syncmap.SyncMap[*redsync.Mutex]
}

type Config struct {
	// The network type, either tcp or unix.
	// Default is tcp.
	Network string

	// string connection url with format redis://{user}:{password}@{host}:{port}/{namespace}
	ConnectionUrl string

	// Maximum number of retries before giving up.
	// Default is 3 retries; -1 (not 0) disables retries.
	MaxRetries int

	// Maximum number of socket connections.
	// Default is 10 connections per every CPU as reported by runtime.NumCPU.
	PoolSize int
}

func New(opt *Config) RedLock {
	u, err := url.Parse(opt.ConnectionUrl)
	if err != nil {
		log.Fatal("Malformed redis connection")
	}

	p, _ := u.User.Password()
	client := goRedisLib.NewClient(&goRedisLib.Options{
		Network:    opt.Network,
		Addr:       u.Host,
		Username:   u.User.Username(),
		Password:   p,
		DB:         0,
		MaxRetries: opt.MaxRetries,
		PoolSize:   opt.PoolSize,
	})
	pool := goredis.NewPool(client)
	redSync := redsync.New(pool)

	ns := strings.TrimPrefix(u.Path, "/")
	if ns == "" {
		ns = defaultNs
	} else {
		ns = ns + "lock:"
	}

	return &redLock{
		redSync:     redSync,
		ns:          ns,
		currentLock: syncmap.NewSyncMap[*redsync.Mutex](),
	}
}

func (r *redLock) AcquireLock(ctx context.Context, key string, attempts int) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RedLock.AcquireLock")
	defer span.End()

	if key == "" {
		return &custerr.ErrChain{
			Message: "[RedLock] Unable to lock for empty key",
			Code:    400,
			Type:    response.ErrBadRequest,
		}
	}

	mutex := r.redSync.NewMutex(r.ns+key, redsync.WithTries(attempts))
	if err := mutex.LockContext(ctx); err != nil {
		return err
	}

	// store mutex lock
	r.currentLock.Store(key, mutex)
	return nil
}

func (r *redLock) AcquireLockWithTTL(ctx context.Context, key string, ttl time.Duration, attempts int) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RedLock.AcquireLockWithTTL")
	defer span.End()

	if key == "" {
		return &custerr.ErrChain{
			Message: "[RedLock] Unable to lock for empty key",
			Code:    400,
			Type:    response.ErrBadRequest,
		}
	}

	mutex := r.redSync.NewMutex(r.ns+key, redsync.WithExpiry(ttl), redsync.WithTries(attempts))
	if err := mutex.LockContext(ctx); err != nil {
		return err
	}

	// store mutex lock
	r.currentLock.Store(key, mutex)
	return nil
}

func (r *redLock) ReleaseLock(ctx context.Context, key string) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RedLock.ReleaseLock")
	defer span.End()

	mutex := r.currentLock.Get(key)
	if mutex == nil {
		return &custerr.ErrChain{
			Message: "[RedLock] Can not find lock to be release",
			Code:    404,
			Type:    response.ErrBadRequest,
		}
	}

	ok, err := mutex.UnlockContext(ctx)
	if !ok || err != nil {
		return &custerr.ErrChain{
			Cause:   err,
			Code:    500,
			Message: "[RedLock] Unable to unlock request",
			Type:    response.ErrBadRequest,
		}
	}

	r.currentLock.Delete(key)
	return nil
}
