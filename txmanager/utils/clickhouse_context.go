package utils

import (
	"context"
	"github.com/nocturna-ta/golib/database/nosql/clickhouse"
)

type clickHouseBatchKey struct {
}

func SetClickHouseBatch(ctx context.Context, batchManager *clickhouse.BatchManager) context.Context {
	ctx = context.WithValue(ctx, clickHouseBatchKey{}, batchManager)
	return ctx
}

func GetClickHouseBatch(ctx context.Context) *clickhouse.BatchManager {
	if ctx == nil {
		return nil
	}

	ctxVal := ctx.Value(clickHouseBatchKey{})
	if ctxVal == nil {
		return nil
	}

	if batchManager, ok := ctxVal.(*clickhouse.BatchManager); ok {
		return batchManager
	}

	return nil
}
