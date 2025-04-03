package mysql

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	libSql "github.com/nocturna-ta/golib/database/sql"
	"github.com/nocturna-ta/golib/event/handler"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/tracing"
	"strings"
	"time"
)

type RetryEventLogRepository struct {
	db *libSql.Store
}

type RetryEventOpts struct {
	DB *libSql.Store
}

func NewRetryEventLogRepository(o *RetryEventOpts) handler.RetryEventLogRepository {
	return &RetryEventLogRepository{
		db: o.DB,
	}
}

const (
	insertRetryLog = `INSERT INTO retry_event_log (id, source_topic, service_name, data, status, count, backoff_date, error, created_time, updated_time)
					  VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	findAllRetryLog = `SELECT id, source_topic, service_name, data, status, count, backoff_date, error, created_time, updated_time
					   FROM retry_event_log
					   WHERE TRUE %s `

	updateRetryLog = `UPDATE retry_event_log SET %s WHERE TRUE %s`
)

func (repo *RetryEventLogRepository) Insert(ctx context.Context, eventLog handler.EventRetryLog) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.Insert")
	defer span.End()

	if eventLog.ID == uuid.Nil {
		eventLog.ID = uuid.New()
	}

	now := time.Now()
	eventLog.CreatedDate = now
	eventLog.UpdatedDate = &now

	_, err := repo.db.GetMaster().ExecContext(ctx, insertRetryLog, eventLog.ID, eventLog.SourceTopic, eventLog.ServiceName, eventLog.Data, eventLog.Status,
		eventLog.Count, eventLog.BackOffDate, eventLog.Error, eventLog.CreatedDate, eventLog.UpdatedDate)

	return err
}

func (repo *RetryEventLogRepository) FindById(ctx context.Context, id uuid.UUID) (*handler.EventRetryLog, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.FindById")
	defer span.End()

	if id == uuid.Nil {
		return nil, fmt.Errorf("empty id")
	}

	var (
		args   []any
		result handler.EventRetryLog
	)

	whereQuery := ` AND id = ?`
	args = append(args, id)

	query := fmt.Sprintf(findAllRetryLog, whereQuery)

	err := repo.db.GetMaster().GetContext(ctx, &result, query, args...)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"id":    id,
		}).ErrorWithCtx(ctx, "[RetryEventLogRepository.FindById] Failed to read record")
		return nil, err
	}

	return &result, nil
}

func (repo *RetryEventLogRepository) FindAll(ctx context.Context, filter *handler.LogFilter) ([]handler.EventRetryLog, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.FindAll")
	defer span.End()

	if filter == nil {
		return nil, fmt.Errorf("empty filter")
	}

	var result []handler.EventRetryLog

	query, args := repo.prepareFindAllQuery(filter)

	err := repo.db.GetMaster().SelectContext(ctx, &result, query, args...)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"filter": filter,
		}).ErrorWithCtx(ctx, "[RetryEventLogRepository.FindAll] Failed to read record")
		return nil, err
	}

	return result, nil
}

func (repo *RetryEventLogRepository) UpdateById(ctx context.Context, id uuid.UUID, eventLog *handler.EventLogToUpdate) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.UpdateById")
	defer span.End()

	if id == uuid.Nil {
		return fmt.Errorf("empty id")
	}

	if eventLog == nil {
		return fmt.Errorf("nil update")
	}

	var (
		err      error
		query    string
		args     []any
		setQuery strings.Builder
	)

	setQuery.WriteString(` updated_time = ?`)
	args = append(args, time.Now())

	if eventLog.Count != nil {
		setQuery.WriteString(`, count=?`)
		args = append(args, *eventLog.Count)
	}
	if eventLog.Status != nil {
		setQuery.WriteString(`, status=?`)
		args = append(args, *eventLog.Status)
	}
	if eventLog.Error != nil {
		setQuery.WriteString(`, error=?`)
		args = append(args, *eventLog.Error)
	}
	if eventLog.BackOffDate != nil {
		setQuery.WriteString(`, backoff_date=?`)
		args = append(args, *eventLog.BackOffDate)
	}

	whereQuery := ` AND id = ?`
	args = append(args, id)

	query = fmt.Sprintf(updateRetryLog, setQuery.String(), whereQuery)

	_, err = repo.db.GetMaster().ExecContext(ctx, query, args...)

	if err != nil {
		log.WithFields(log.Fields{
			"error":        err,
			"kafka-log-id": id,
			"request":      eventLog,
		}).ErrorWithCtx(ctx, "[RetryEventLogRepository.UpdateById] Failed to update retry event log")
		return err
	}

	return nil
}

func (repo *RetryEventLogRepository) prepareFindAllQuery(filter *handler.LogFilter) (string, []any) {
	var (
		sb   strings.Builder
		args []any
	)

	if filter.MaxRetry > 0 {
		sb.WriteString(` AND count <= ?`)
		args = append(args, filter.MaxRetry)
	}

	if len(filter.InStatus) > 0 {
		sb.WriteString(` AND status IN (?`)
		sb.WriteString(strings.Repeat(`,?`, len(filter.InStatus)-1))
		for _, status := range filter.InStatus {
			args = append(args, status)
		}
		sb.WriteString(`)`)
	}

	if filter.MaxBackOffDate != nil {
		sb.WriteString(` AND backoff_date <= ?`)
		args = append(args, *filter.MaxBackOffDate)
	}

	if filter.SortBy != "" {
		sb.WriteString(` ORDER BY ` + filter.GetSortBy() + ` ` + filter.GetSortDir())
	}

	if filter.Limit > 0 {
		sb.WriteString(` LIMIT ? OFFSET ?`)
		args = append(args, filter.GetLimit())
		args = append(args, filter.GetSkip())
	}

	return fmt.Sprintf(findAllRetryLog, sb.String()), args
}
