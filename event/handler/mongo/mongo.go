package mongo

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/nocturna-ta/golib/event/handler"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/tracing"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type RetryEventLogRepository struct {
	collection *mongo.Collection
}

type RetryEventOpts struct {
	DB *mongo.Database
}

func NewRetryEventLogRepository(opts *RetryEventOpts) handler.RetryEventLogRepository {
	return &RetryEventLogRepository{
		collection: opts.DB.Collection("retry_event"),
	}
}

func (r *RetryEventLogRepository) Insert(ctx context.Context, eventLog handler.EventRetryLog) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.Insert")
	defer span.End()

	if eventLog.ID == uuid.Nil {
		eventLog.ID = uuid.New()
	}
	eventLog.CreatedDate = time.Now()
	eventLog.UpdatedDate = &eventLog.CreatedDate

	_, err := r.collection.InsertOne(ctx, eventLog)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"log":   eventLog,
		}).ErrorWithCtx(ctx, "[RetryEventLogRepository.Insert] Failed to insert retry log")
		return err
	}

	return nil
}

func (r *RetryEventLogRepository) FindById(ctx context.Context, id uuid.UUID) (*handler.EventRetryLog, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.FindById")
	defer span.End()

	filter := bson.D{
		{Key: "_id", Value: id},
	}

	eventLog := &handler.EventRetryLog{}

	err := r.collection.FindOne(ctx, filter).Decode(eventLog)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"log-id": id,
		}).ErrorWithCtx(ctx, "[RetryEventLogRepository.FindById] Failed to find retry log")
		return nil, err
	}

	return eventLog, nil
}

func (r *RetryEventLogRepository) FindAll(ctx context.Context, filter *handler.LogFilter) ([]handler.EventRetryLog, error) {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.FindAll")
	defer span.End()

	filter.Validate()

	queryFilter := bson.D{}

	if filter.MaxRetry > 0 {
		queryFilter = append(queryFilter, bson.E{Key: "count", Value: bson.D{
			{Key: "$lte", Value: filter.MaxRetry},
		}})
	}
	if len(filter.InStatus) > 0 {
		queryFilter = append(queryFilter, bson.E{Key: "status", Value: bson.D{
			{Key: "$in", Value: filter.InStatus},
		}})
	}
	if filter.MaxBackOffDate != nil {
		queryFilter = append(queryFilter, bson.E{Key: "backoff_date", Value: bson.D{
			{Key: "$lte", Value: *filter.MaxBackOffDate},
		}})
	}

	opts := options.Find()
	if filter.SortBy != "" {
		opts.SetSort(bson.D{
			{Key: filter.GetSortBy(), Value: filter.GetSortDir()},
		})
	}
	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.GetLimit()))
		opts.SetSkip(int64(filter.GetSkip()))
	}

	result, err := r.collection.Find(ctx, queryFilter, opts)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"filter": filter,
		}).WarnWithCtx(ctx, "[RetryEventLogRepository.FindAll] Failed to FindAll retry event log")
		return nil, err
	}

	var logs []handler.EventRetryLog

	err = result.All(ctx, &logs)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"filter": filter,
		}).WarnWithCtx(ctx, "[RetryEventLogRepository.FindAll] Failed to decode retry event log")
		return nil, err
	}

	return logs, nil
}

func (r *RetryEventLogRepository) UpdateById(ctx context.Context, id uuid.UUID, eventLog *handler.EventLogToUpdate) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "RetryEventLogRepository.UpdateById")
	defer span.End()

	if id == uuid.Nil {
		return fmt.Errorf("empty id")
	}

	if eventLog == nil {
		return fmt.Errorf("nil update")
	}

	updateFields := bson.D{}
	if eventLog.Count != nil {
		updateFields = append(updateFields, bson.E{Key: "count", Value: *eventLog.Count})
	}
	if eventLog.Status != nil {
		updateFields = append(updateFields, bson.E{Key: "status", Value: *eventLog.Status})
	}
	if eventLog.BackOffDate != nil {
		updateFields = append(updateFields, bson.E{Key: "backoff_date", Value: *eventLog.BackOffDate})
	}
	if eventLog.Error != nil {
		updateFields = append(updateFields, bson.E{Key: "error", Value: *eventLog.Error})
	}

	update := bson.D{
		{Key: "$set", Value: updateFields},
	}

	_, err := r.collection.UpdateByID(ctx, id, update)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"id":      id,
			"request": eventLog,
		}).ErrorWithCtx(ctx, "[RetryEventLogRepository.UpdateById] Failed to update kafka log")
		return err
	}

	return nil
}
