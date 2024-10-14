package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/nocturna-ta/golib/cache/redlock"
	libCtx "github.com/nocturna-ta/golib/context"
	"github.com/nocturna-ta/golib/event"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/utils/syncmap"
	"github.com/panjf2000/ants"

	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	lockRetryEventKey = "retry-event-lock:"
	lockRetryTTL      = time.Hour
)

type (
	EventRetry struct {
		eventLogRepo RetryEventLogRepository
		publisher    event.MessagePublisher
		redLock      redlock.RedLock
		retryQueue   *syncmap.SyncMap[scheduleRetryLog]
		retryConfig  EventRetryConfig
	}

	scheduleRetryLog struct {
		EventLog    EventRetryLog
		IsScheduled bool
	}

	EventRetryConfig struct {
		PoolSize  int
		DelayLoad time.Duration
		MaxRetry  int
	}

	RetryOptions struct {
		EventLogRepo RetryEventLogRepository
		Publisher    event.MessagePublisher
		RedLock      redlock.RedLock
		RetryConfig  EventRetryConfig
	}
)

func NewEventRetry(opt *RetryOptions) *EventRetry {
	if opt.RetryConfig.PoolSize < 1 {
		opt.RetryConfig.PoolSize = 3
	}

	return &EventRetry{
		eventLogRepo: opt.EventLogRepo,
		publisher:    opt.Publisher,
		redLock:      opt.RedLock,
		retryQueue:   syncmap.NewSyncMap[scheduleRetryLog](),
		retryConfig:  opt.RetryConfig,
	}
}

func (er *EventRetry) Run() {
	defer ants.Release()

	wg := new(sync.WaitGroup)

	pool, err := er.initializePublisherPool(wg)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatalf("Failed to initialize publisher pool")
	}
	defer pool.Release()

	go er.loadQueue()

	isClosed := false

	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		<-signalCh
		isClosed = true
	}()

	for !isClosed {
		allKeys := er.retryQueue.GetAllKey()
		for _, key := range allKeys {
			queue := er.retryQueue.Get(key)
			if queue.EventLog.ID != uuid.Nil {
				queue.IsScheduled = true
				er.retryQueue.Store(key, queue)

				wg.Add(1)
				err = pool.Invoke(queue.EventLog)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err,
					}).Error("Failed to invoke publisher function")
				}
			}
		}

		time.Sleep(er.retryConfig.DelayLoad)
	}

	wg.Wait()
}

func (er *EventRetry) initializePublisherPool(wg *sync.WaitGroup) (*ants.PoolWithFunc, error) {
	p, err := ants.NewPoolWithFunc(er.retryConfig.PoolSize, func(val any) {
		requestId := uuid.New().String()
		ctx := context.WithValue(context.Background(), libCtx.RequestIdKey, requestId)

		log.WithFields(log.Fields{
			"input-value": val,
		}).InfoWithCtx(ctx, "[poolFunction] Trying to publish retry topic")

		eventRetryLog, ok := val.(EventRetryLog)
		if ok {
			waitTime := time.Since(eventRetryLog.BackOffDate)
			if waitTime < 0 {
				time.Sleep(-1 * waitTime)
			}

			err := er.publishRetryTopic(ctx, eventRetryLog)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).ErrorWithCtx(ctx, "[poolFunction] Failed to publish retry topic")
			} else {
				log.WithFields(log.Fields{
					"value": val,
				}).InfoWithCtx(ctx, "[poolFunction] Success publish retry topic")
			}
		} else {
			log.WithFields(log.Fields{
				"value": val,
			}).InfoWithCtx(ctx, "[poolFunction] Failed to publish retry topic because invalid cast type")
		}
		wg.Done()
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("[initializePublisherPool] Failed to initialize publisher pool")
		return nil, err
	}
	return p, nil
}

func (er *EventRetry) loadQueue() {
	for true {
		requestId := uuid.New().String()
		ctx := context.WithValue(context.Background(), libCtx.RequestIdKey, requestId)

		backOffTime := time.Now().Add(er.retryConfig.DelayLoad)
		retryLogs, err := er.eventLogRepo.FindAll(ctx, &LogFilter{
			MaxRetry: er.retryConfig.MaxRetry,
			InStatus: []LogStatus{
				Failed,
			},
			MaxBackOffDate: &backOffTime,
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).ErrorWithCtx(ctx, "[eventRetry/loadQueue] Failed to read retry record")
		}

		for _, retryLog := range retryLogs {
			existingQ := er.retryQueue.Get(retryLog.ID.String())
			if existingQ.EventLog.ID == uuid.Nil {
				scheduledLog := scheduleRetryLog{
					EventLog:    retryLog,
					IsScheduled: false,
				}
				er.retryQueue.Store(retryLog.ID.String(), scheduledLog)
			}
		}

		sleepTime := time.Since(backOffTime)
		time.Sleep(-1 * sleepTime)
	}
}

func (er *EventRetry) publishRetryTopic(ctx context.Context, eventRetryLog EventRetryLog) error {
	existing, err := er.eventLogRepo.FindById(ctx, eventRetryLog.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"retry_event_log": eventRetryLog,
		}).ErrorWithCtx(ctx, "[publishRetryTopic] Failed to get existing log")
		return err
	}

	if !existing.IsAllowRetry(er.retryConfig.MaxRetry) {
		return nil
	}

	retryStatus := WaitingRetry
	err = er.eventLogRepo.UpdateById(ctx, existing.ID, &EventLogToUpdate{
		Status: &retryStatus,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"retry_event_log": existing,
		}).ErrorWithCtx(ctx, "[publishRetryTopic] Failed to update retry log")
		return err
	}

	attempt := existing.Count
	topicName := GetBackOffTopic(existing.SourceTopic, attempt)

	lockKey := lockRetryEventKey + existing.ID.String() + ":" + strconv.Itoa(existing.Count)
	if err = er.redLock.AcquireLockWithTTL(ctx, lockKey, lockRetryTTL, 1); err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"attempt":         attempt,
			"retry_event_log": existing,
		}).ErrorWithCtx(ctx, "[publishRetryTopic] Not allow to publish")
		return err
	}

	retryLog := &BackoffRetry{}
	_ = json.Unmarshal(existing.Data, &retryLog)

	data := map[string]any{}
	_ = json.Unmarshal(retryLog.Data, &data)

	ctx = er.newContextFromMetadata(ctx, retryLog.Metadata)

	err = er.publisher.Publish(ctx, topicName, existing.ID.String(), data, map[string]any{
		libCtx.MetadataLogRefId:      existing.ID,
		libCtx.MetadataRetryAttempts: attempt,
		"source-topic":               retryLog.SourceTopic,
		"last-metadata":              retryLog.Metadata,
		"last-error":                 retryLog.Error,
	})
	if err == nil {
		// remove from queue
		er.retryQueue.Delete(eventRetryLog.ID.String())

		return nil
	}

	log.WithFields(log.Fields{
		"error":           err,
		"retry_event_log": existing,
	}).ErrorWithCtx(ctx, "[RetryModule.publishRetryTopic] Failed to send retry topic")

	// trying to update kafka log
	failedStatus := Failed
	errStr := err.Error()
	err = er.eventLogRepo.UpdateById(ctx, existing.ID, &EventLogToUpdate{
		Status: &failedStatus,
		Error:  &errStr,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"retry_event_log": existing,
		}).ErrorWithCtx(ctx, "[publishRetryTopic] Failed to update kafka log")
	}

	// remove from queue
	er.retryQueue.Delete(eventRetryLog.ID.String())

	return fmt.Errorf("failed to publish retry topic %s with log_id=%s", existing.SourceTopic, existing.ID)
}

func (er *EventRetry) newContextFromMetadata(ctx context.Context, metadata map[string]any) context.Context {
	reqCtx := libCtx.RequestContext{}

	if val, ok := metadata[libCtx.XRequestId]; ok {
		if valStr, ok := val.(string); ok {
			reqCtx.RequestId = valStr
		}
	}

	if val, ok := metadata[libCtx.XUserId]; ok {
		if valStr, ok := val.(string); ok {
			reqCtx.UserId = valStr
		}
	}

	if val, ok := metadata[libCtx.XChannelId]; ok {
		if valStr, ok := val.(string); ok {
			reqCtx.ChannelId = valStr
		}
	}

	if val, ok := metadata[libCtx.XAccountId]; ok {
		if valStr, ok := val.(string); ok {
			reqCtx.AccountId = valStr
		}
	}

	ctx = context.WithValue(ctx, libCtx.RequestContextKey, reqCtx)

	return ctx
}
