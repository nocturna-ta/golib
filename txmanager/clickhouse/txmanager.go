package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"github.com/nocturna-ta/golib/database/nosql/clickhouse"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/txmanager"
)

func init() {
	txmanager.Register("clickhouse", NewTxManager)
}

type (
	manager struct {
		client clickhouse.Client
	}

	Config struct {
		Client clickhouse.Client
	}
)

func NewTxManager(_ context.Context, config any) (txmanager.TxManager, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, fmt.Errorf("failed to decode config")
	}

	if cfg.Client == nil {
		return nil, fmt.Errorf("clickhouse client is required")
	}

	return &manager{client: cfg.Client}, nil
}

func (m *manager) Execute(ctx context.Context, fn txmanager.TxFn, opts any) (result any, err error) {

	defer func() {
		if p := recover(); p != nil {
			log.WithFields(log.Fields{
				"panic": p,
			}).ErrorWithCtx(ctx, "Panic when executing ClickHouse batch operation")
			err = errors.New("panic happened when executing batch operation: " + fmt.Sprintf("%v", p))
		}
	}()

	result, err = fn(ctx)

	return result, err
}

// BatchContext holds the batch operation context
type BatchContext struct {
	ctx    context.Context
	client clickhouse.Client
	batch  any // This would be the actual batch from ClickHouse driver
}

func SetBatchContext(ctx context.Context, client clickhouse.Client, batch any) context.Context {
	return context.WithValue(ctx, batchContextKey{}, &BatchContext{
		ctx:    ctx,
		client: client,
		batch:  batch,
	})
}

func GetBatchContext(ctx context.Context) *BatchContext {
	if ctx == nil {
		return nil
	}

	ctxVal := ctx.Value(batchContextKey{})
	if ctxVal == nil {
		return nil
	}

	if batchCtx, ok := ctxVal.(*BatchContext); ok {
		return batchCtx
	}

	return nil
}

type batchContextKey struct{}

type BatchManager struct {
	client clickhouse.Client
}

func NewBatchManager(client clickhouse.Client) *BatchManager {
	return &BatchManager{client: client}
}

func (bm *BatchManager) ExecuteBatch(ctx context.Context, query string, fn func(batch *clickhouse.Batch) error) error {
	batch, err := bm.client.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	if err := fn(batch); err != nil {
		if abortErr := batch.Abort(); abortErr != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"abort_error": abortErr,
			}).ErrorWithCtx(ctx, "Failed to abort batch after error")
		}
		return err
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}
