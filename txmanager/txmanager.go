package txmanager

import (
	"context"
	"errors"
	"golib/tracing"
)

var (
	managers = map[string]Factory{}
)

type (
	Manager struct {
		txManager TxManager
	}

	DriverConfig struct {
		Type   string `json:"type" mapstructure:"type"`
		Config any    `json:"config" mapstructure:"config"`
	}

	TxFn    func(ctx context.Context) (any, error)
	Factory func(ctx context.Context, config any) (TxManager, error)
)

type (
	TxManager interface {
		Execute(ctx context.Context, fn TxFn, opts any) (any, error)
	}
)

func Register(name string, factory Factory) {
	managers[name] = factory
}

func New(ctx context.Context, cfg *DriverConfig) (*Manager, error) {
	if cfg == nil {
		return nil, errors.New("[txmanager] missing config")
	}

	tf, ok := managers[cfg.Type]
	if !ok {
		return nil, errors.New("[txmanager] unsupported driver")
	}

	mgr, err := tf(ctx, cfg.Config)
	if err != nil {
		return nil, err
	}

	return &Manager{txManager: mgr}, nil
}

func (m *Manager) Execute(ctx context.Context, fn TxFn, opts any) (any, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "TxManager.Execute")
	defer span.End()

	return m.txManager.Execute(ctx, fn, opts)
}
