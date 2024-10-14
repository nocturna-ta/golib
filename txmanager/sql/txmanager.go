package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	libSql "github.com/nocturna-ta/golib/database/sql"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/txmanager"
	"github.com/nocturna-ta/golib/txmanager/utils"
)

func init() {
	txmanager.Register("sql", NewTxManager)
}

type (
	manager struct {
		db *libSql.Store
	}

	Config struct {
		DB *libSql.Store
	}
)

func NewTxManager(_ context.Context, config any) (txmanager.TxManager, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, fmt.Errorf("failed to decode config")
	}

	return &manager{db: cfg.DB}, nil
}

func (m *manager) Execute(ctx context.Context, fn txmanager.TxFn, opts any) (result any, err error) {
	var txOpts *sql.TxOptions

	if opts != nil {
		opt, ok := opts.(*sql.TxOptions)
		if !ok {
			return nil, fmt.Errorf("options is not valid")
		}
		txOpts = opt
	}

	sqlTx, err := m.db.GetMaster().BeginTxx(ctx, txOpts)
	if err != nil {
		return nil, err
	}

	txCtx := utils.SetSqlTx(ctx, sqlTx)

	defer func() {
		if p := recover(); p != nil {
			// a panic occurred, rollback and return error
			sqlTx.Rollback() //nolint:errcheck
			log.WithFields(log.Fields{
				"panic": p,
			}).ErrorWithCtx(ctx, "Panic when executing transaction")
			err = errors.New("panic happened when executing transaction because: " + fmt.Sprintf("%v", p))
		} else if err != nil {
			// error occurred, rollback
			log.WithFields(log.Fields{
				"error": err,
			}).ErrorWithCtx(ctx, "Error when executing transaction")
			sqlTx.Rollback() //nolint:errcheck
		} else {
			// all good, commit
			err = sqlTx.Commit()
		}
	}()

	result, err = fn(txCtx)
	return result, err
}
