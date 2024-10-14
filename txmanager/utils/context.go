package utils

import (
	"context"
	"github.com/jmoiron/sqlx"
)

type ctxKey struct{}

func SetSqlTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	ctx = context.WithValue(ctx, ctxKey{}, tx)
	return ctx
}

func GetSqlTx(ctx context.Context) *sqlx.Tx {
	if ctx == nil {
		return nil
	}

	ctxVal := ctx.Value(ctxKey{})
	if ctxVal == nil {
		return nil
	}

	if sqlTx, ok := ctxVal.(*sqlx.Tx); ok {
		return sqlTx
	}

	return nil
}
