package mongodb

import (
	"context"
	"fmt"
	"github.com/nocturna-ta/golib/txmanager"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func init() {
	txmanager.Register("mongodb", NewTxManager)
}

type (
	manager struct {
		db *mongo.Database
	}

	Config struct {
		DB *mongo.Database
	}
)

func NewTxManager(_ context.Context, config any) (txmanager.TxManager, error) {
	cfg, ok := config.(Config)
	if !ok {
		return nil, fmt.Errorf("failed to decode config")
	}

	return &manager{db: cfg.DB}, nil
}

func (m *manager) Execute(ctx context.Context, fn txmanager.TxFn, opts any) (any, error) {
	var txOpts *options.SessionOptions

	if opts != nil {
		opt, ok := opts.(*options.SessionOptions)
		if !ok {
			return nil, fmt.Errorf("options is not valid")
		}
		txOpts = opt
	}

	session, err := m.db.Client().StartSession(txOpts)
	defer session.EndSession(ctx)
	if err != nil {
		return nil, err
	}

	res, err := session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (any, error) {
		return fn(sessCtx)
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
