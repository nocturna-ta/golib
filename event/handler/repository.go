package handler

import (
	"context"
	"github.com/google/uuid"
)

type RetryEventLogRepository interface {
	Insert(ctx context.Context, log EventRetryLog) error
	FindById(ctx context.Context, id uuid.UUID) (*EventRetryLog, error)
	FindAll(ctx context.Context, filter *LogFilter) ([]EventRetryLog, error)
	UpdateById(ctx context.Context, id uuid.UUID, log *EventLogToUpdate) error
}
