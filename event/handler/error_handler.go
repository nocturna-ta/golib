package handler

import (
	"context"
	"encoding/json"
	"github.com/avast/retry-go/v4"
	"github.com/google/uuid"
	libCtx "github.com/nocturna-ta/golib/context"
	"github.com/nocturna-ta/golib/event"
	"github.com/nocturna-ta/golib/log"

	"time"
)

type ErrorHandlerLevel string

const (
	ErrorHandlerPhase1 ErrorHandlerLevel = "Phase1"
	ErrorHandlerPhase2 ErrorHandlerLevel = "Phase2"
	NoErrorHandler     ErrorHandlerLevel = "NoErrorHandler"
)

func (e *EventHandler) handlePhase1(ctx context.Context, handler event.ConsumerHandler, message *event.EventConsumeMessage, withBackOff bool) error {
	var handlerError error
	var retryErr retry.Error

	err := retry.Do(
		func() error {
			if err := handler(ctx, message); err != nil {
				log.WithFields(log.Fields{
					"error":    err,
					"topic":    message.Topic,
					"key":      message.Key,
					"metadata": message.Metadata,
					"data":     string(message.Data),
				}).WarnWithCtx(ctx, "[event-handler/phase1] Error while executing event consumer handler")
				handlerError = err
				return err
			}
			return nil
		},
		retry.Attempts(uint(e.retryCfg.MaxRetry)),
		retry.DelayType(setDelayType(withBackOff, e.retryCfg.MaxJitter)),
		retry.Delay(e.retryCfg.RetryInitialDelay),
		retry.MaxJitter(e.retryCfg.MaxJitter),
	)

	if err != nil {
		retryErr = append(retryErr, handlerError)
		e.sendDlq(ctx, message, retryErr)
	}

	return nil
}

func (e *EventHandler) sendDlq(ctx context.Context, message *event.EventConsumeMessage, err error) {
	data := map[string]any{}
	json.Unmarshal(message.Data, &data)

	dlqMsg := DlqMessage{
		Data:        data,
		SourceTopic: message.Topic,
		Metadata:    message.Metadata,
		ServiceName: e.serviceName,
		Timestamp:   time.Now(),
		Error:       err.Error(),
	}

	err = e.publisher.publisher.Publish(ctx, e.publisher.dlqTopic, "", dlqMsg, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"dlq_data": dlqMsg,
			"topic":    message.Topic,
			"key":      message.Key,
			"metadata": message.Metadata,
			"data":     data,
		}).ErrorWithCtx(ctx, "Error while sending to dlq topic")
	}
}

func (e *EventHandler) handlePhase2(ctx context.Context, handler event.ConsumerHandler, message *event.EventConsumeMessage, withBackOff bool) error {
	err := handler(ctx, message)

	if err != nil {
		attempts := 1
		if val, ok := message.Metadata[libCtx.MetadataRetryAttempts]; ok {
			if valFloat, ok := val.(float64); ok {
				attempts = int(valFloat)
				attempts++
			}
		}

		if attempts > e.retryCfg.MaxRetry {
			e.sendDlq(ctx, message, err)
			return err
		}

		if withBackOff {
			e.sendRetryBackOffEvent(ctx, message, err, attempts)
		} else {
			e.sendRetryEvent(ctx, message, err, attempts)
		}
	}

	return nil
}

func (e *EventHandler) sendRetryBackOffEvent(ctx context.Context, message *event.EventConsumeMessage, handlerErr error, attempts int) {
	backOff := e.retryCfg.BackOffConfig[attempts]
	errTrace := handlerErr.Error()

	var (
		refId       string
		referenceId uuid.UUID
		existingLog *EventRetryLog
		retryData   BackoffRetry
		err         error
	)

	if val, ok := message.Metadata[libCtx.MetadataLogRefId]; ok {
		refId = val.(string)
	}

	if refId != "" {
		referenceId, err = uuid.Parse(refId)
		if err == nil {
			existingLog, err = e.retryLogRepo.FindById(ctx, referenceId)
			if err == nil {
				failedStatus := Failed
				backOffDate := time.Now().Add(backOff)
				err = e.retryLogRepo.UpdateById(ctx, existingLog.ID, &EventLogToUpdate{
					Status:      &failedStatus,
					Count:       &attempts,
					BackOffDate: &backOffDate,
					Error:       &errTrace,
				})
				if err == nil {
					return
				}
			}
		}
	}

	retryData = BackoffRetry{
		SourceTopic: message.Topic,
		Data:        message.Data,
		Metadata:    message.Metadata,
		Timestamp:   time.Now(),
		Error:       errTrace,
	}
	payload, _ := json.Marshal(retryData)

	eventLog := EventRetryLog{
		ID:          uuid.New(),
		SourceTopic: message.Topic,
		ServiceName: e.serviceName,
		Data:        payload,
		Status:      Failed,
		Count:       attempts,
		BackOffDate: time.Now().Add(backOff),
		Error:       &errTrace,
	}

	err = e.retryLogRepo.Insert(ctx, eventLog)
	if err != nil {
		log.WithFields(log.Fields{
			"error":           err,
			"retry_event_log": eventLog,
			"topic":           message.Topic,
			"key":             message.Key,
			"metadata":        message.Metadata,
			"data":            string(message.Data),
		}).ErrorWithCtx(ctx, "[EventHandler.sendRetryBackOffEvent] Error while inserting to retry event log")
	}
}

func (e *EventHandler) sendRetryEvent(ctx context.Context, message *event.EventConsumeMessage, handlerErr error, attempts int) {
	var tempError string

	if handlerErr != nil {
		tempError = handlerErr.Error()
	}

	topic := GetBackOffTopic(message.Topic, attempts)

	data := map[string]any{}
	json.Unmarshal(message.Data, &data)

	err := e.publisher.publisher.Publish(ctx, topic, message.Key, data, map[string]any{
		libCtx.MetadataRetryAttempts: attempts,
		"source-topic":               message.Topic,
		"last-metadata":              message.Metadata,
		"last-error":                 tempError,
	})

	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"topic":    message.Topic,
			"key":      message.Key,
			"metadata": message.Metadata,
			"data":     string(message.Data),
		}).ErrorWithCtx(ctx, "[EventHandler.sendRetryEvent] Error while sending to retry topic")
	}
}

func setDelayType(withBackOff bool, maxJitter time.Duration) func(n uint, err error, config *retry.Config) time.Duration {
	return func(n uint, err error, config *retry.Config) time.Duration {
		if maxJitter > time.Millisecond {
			return retry.RandomDelay(n, err, config)
		}
		if withBackOff {
			return retry.BackOffDelay(n, err, config)
		}
		return retry.FixedDelay(n, err, config)
	}
}
