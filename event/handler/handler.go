package handler

import (
	"context"
	"fmt"
	"golib/event"
	"golib/log"
	"runtime/debug"
	"sort"
	"time"
)

type (
	EventHandler struct {
		serviceName  string
		retryCfg     eventRetryConfig
		publisher    publisher
		retryLogRepo RetryEventLogRepository
	}

	eventRetryConfig struct {
		MaxRetry          int
		RetryInitialDelay time.Duration
		MaxJitter         time.Duration // set MaxJitter > 1ms will enable random delay
		HandlerTimeout    time.Duration
		BackOffConfig     map[int]time.Duration
		MaxRetryBackOff   int
	}

	publisher struct {
		publisher event.MessagePublisher
		dlqTopic  string
	}

	Options struct {
		RetryConfig RetryConfig
		Publisher   event.MessagePublisher
		DlqTopic    string
		LogRepo     RetryEventLogRepository
		ServiceName string
	}

	RetryConfig struct {
		MaxRetry          int
		RetryInitialDelay time.Duration
		MaxJitter         time.Duration
		HandlerTimeout    time.Duration
		BackOffConfig     []time.Duration
		MaxBackoffAttempt int
	}
)

func New(opts *Options) EventHandler {
	backOffAttempt := DefaultMaxBackoffAttempt
	if opts.RetryConfig.MaxBackoffAttempt > 0 {
		backOffAttempt = opts.RetryConfig.MaxBackoffAttempt
	}

	backOffCfgFunc := func(cfg RetryConfig) (map[int]time.Duration, int) {
		backOff := map[int]time.Duration{}
		maxRetry := len(cfg.BackOffConfig)

		sort.Slice(cfg.BackOffConfig, func(i, j int) bool {
			return cfg.BackOffConfig[i] < cfg.BackOffConfig[j]
		})

		for i, duration := range cfg.BackOffConfig {
			backOff[i+1] = duration
			if i > backOffAttempt {
				maxRetry = backOffAttempt
				break
			}
		}

		return backOff, maxRetry
	}

	backOffCfg, maxRetry := backOffCfgFunc(opts.RetryConfig)

	return EventHandler{
		retryCfg: eventRetryConfig{
			MaxRetry:          opts.RetryConfig.MaxRetry,
			RetryInitialDelay: opts.RetryConfig.RetryInitialDelay,
			MaxJitter:         opts.RetryConfig.MaxJitter,
			HandlerTimeout:    opts.RetryConfig.HandlerTimeout,
			BackOffConfig:     backOffCfg,
			MaxRetryBackOff:   maxRetry,
		},
		publisher: publisher{
			publisher: opts.Publisher,
			dlqTopic:  opts.DlqTopic,
		},
		retryLogRepo: opts.LogRepo,
		serviceName:  opts.ServiceName,
	}
}

func (e *EventHandler) HandleConsume(ctx context.Context, message *event.EventConsumeMessage, errorHandler string, handler event.ConsumerHandler, withBackOff bool) (err error) {
	ctx, cancel := context.WithTimeout(ctx, e.retryCfg.HandlerTimeout)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			stackTrace := string(debug.Stack())

			log.WithFields(log.Fields{
				"handler":     handler,
				"stack-trace": stackTrace,
				"error":       fmt.Sprintf("%+v", err),
			}).ErrorWithCtx(ctx, "[consumer.panicHandler] panic have occurred")

			if errR, ok := r.(error); !ok {
				err = fmt.Errorf(`[consumer] error when executing handler %+v`, errR)
			}
		}

	}()

	switch ErrorHandlerLevel(errorHandler) {
	case NoErrorHandler:
		err = handler(ctx, message)
	case ErrorHandlerPhase1:
		err = e.handlePhase1(ctx, handler, message, withBackOff)
	case ErrorHandlerPhase2:
		err = e.handlePhase2(ctx, handler, message, withBackOff)
	default:
		// default no error handler
		err = handler(ctx, message)
	}

	return err
}
