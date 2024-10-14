package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/newrelic/go-agent/v3/newrelic"
	libCtx "github.com/nocturna-ta/golib/context"
	"github.com/nocturna-ta/golib/log"
	"github.com/nocturna-ta/golib/tracing"

	newrelicLib "github.com/nocturna-ta/golib/tracing/newrelic"
	sentryLib "github.com/nocturna-ta/golib/tracing/sentry"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	stop  uint32 = 0
	start uint32 = 1

	DefaultConsumerWorkers = 1
)

var (
	consumers = map[string]ConsumerFactory{}
)

type (
	Consumer struct {
		subscriber           Subscriber
		subscriberWorkerPool []SubscriberWorkerPool
		eventConfig          *EventConfig
		workerPoolConfig     *WorkerPoolConfig
		commitStrategy       CommitStrategy
		running              uint32
		lock                 sync.Mutex
		stopch               chan bool
		shutdown             chan bool
		newRelic             *newrelic.Application
		eventHandler         ConsumerEventHandler
	}

	EventConsumeMessage struct {
		Topic    string
		Key      string
		Metadata map[string]any
		Data     []byte
	}

	eventConsumeMessageRead struct {
		Data     json.RawMessage `json:"data" mapstructure:"data"`
		Metadata map[string]any  `json:"metadata,omitempty" mapstructure:"metadata"`
	}

	ConsumerConfig struct {
		Consumer         *DriverConfig        `json:"consumer" mapstructure:"consumer"`
		EventConfig      *EventConfig         `json:"event_config" mapstructure:"event_config"`
		WorkerPoolConfig *WorkerPoolConfig    `json:"worker_pool_config" mapstructure:"worker_pool_config"`
		CommitStrategy   *DriverConfig        `json:"commit_strategy" mapstructure:"commit_strategy"`
		EventHandler     ConsumerEventHandler `json:"event_handler" mapstructure:"event_handler"`
		NewRelicOpts     *newrelicLib.Options `json:"new_relic_opts" mapstructure:"new_relic_opts"`
		SentryConfig     *sentryLib.Config    `json:"sentry_config" mapstructure:"sentry_config"`
	}

	SubscriberWorkerPool struct {
		workers        int
		consumerGroup  ConsumerGroup
		handler        ConsumerHandler
		topic          string
		group          string
		commitStrategy CommitStrategy
		newRelic       *newrelic.Application
	}

	WorkerPoolConfig map[string]any
	ConsumerFactory  func(ctx context.Context, config any) (Subscriber, error)
	Job              func(ctx context.Context) error
	TopicName        string

	ConsumerHandlerConfig struct {
		ConsumerGroup     string
		ErrorHandlerLevel string
		Handler           ConsumerHandler
		WithBackOff       bool
	}
)

type (
	ConsumeMessage interface {
		GetEventConsumeMessage(ctx context.Context) (*EventConsumeMessage, error)
		Commit(ctx context.Context) error
	}

	Subscriber interface {
		Register(ctx context.Context, topic string, group string) (ConsumerGroup, error)
	}

	ConsumerEventHandler interface {
		HandleConsume(ctx context.Context, message *EventConsumeMessage, errorHandler string, handler ConsumerHandler, withBackOff bool) error
	}

	Closer interface {
		Close() error
	}

	ConsumerGroup interface {
		Start(ctx context.Context) error
		GetMessage(ctx context.Context) (ConsumeMessage, error)
	}
)

func RegisterConsumer(name string, factory ConsumerFactory) {
	consumers[name] = factory
}

func NewConsumer(ctx context.Context, config *ConsumerConfig) (*Consumer, error) {
	if config == nil {
		return nil, errors.New("[event/consumer] missing config")
	}

	consumer := &Consumer{
		subscriberWorkerPool: make([]SubscriberWorkerPool, 0),
		eventConfig:          &EventConfig{},
		workerPoolConfig:     &WorkerPoolConfig{},
		commitStrategy:       CommitOnSuccessStrategy,
		running:              stop,
		lock:                 sync.Mutex{},
		stopch:               make(chan bool, 2),
	}

	if config.EventConfig != nil {
		consumer.eventConfig = config.EventConfig
	}

	if config.WorkerPoolConfig != nil {
		consumer.workerPoolConfig = config.WorkerPoolConfig
	}

	if config.CommitStrategy != nil {
		strategyFactory, ok := commitStrategy[config.CommitStrategy.Type]
		if !ok {
			return nil,
				fmt.Errorf("[event/consumer] invalid consume strategy [%s]", config.CommitStrategy)
		}

		strategy, err := strategyFactory(ctx, config.CommitStrategy.Config)
		if err != nil {
			return nil,
				fmt.Errorf("[event/consumer] failed to init consumerstrategy [%s]: %w",
					config.CommitStrategy.Type, err)
		}

		consumer.commitStrategy = strategy
	}

	if config.EventHandler != nil {
		consumer.eventHandler = config.EventHandler
	}

	if config.Consumer == nil {
		return nil, errors.New("[event/consumer] missing consumer driver config")
	}

	factory, ok := consumers[config.Consumer.Type]
	if !ok {
		return nil, fmt.Errorf("[event/consumer] unsupported consumer driver: %s",
			config.Consumer.Type)
	}

	if config.NewRelicOpts != nil {
		consumer.newRelic = newrelicLib.SetupNewRelic(config.NewRelicOpts)
	}

	if config.SentryConfig != nil {
		_ = sentryLib.Init(config.SentryConfig)
	}

	subscriber, err := factory(ctx, config.Consumer.Config)
	if err != nil {
		return nil, err
	}

	consumer.subscriber = subscriber

	return consumer, nil
}

func (c *Consumer) isRunning() bool {
	return atomic.LoadUint32(&c.running) == start
}

// SetCommitStrategy set commit strategy for this consumer
func (c *Consumer) SetCommitStrategy(strategy CommitStrategy) {
	c.commitStrategy = strategy
}

func (c *Consumer) Subscribe(ctx context.Context, topic, group string, handler ConsumerHandler) error {
	if c.isRunning() {
		return fmt.Errorf("[event/consumer] consumer already started, can't register new subscriber")
	}

	topic = c.eventConfig.getTopic(topic)

	consumerGroup, err := c.subscriber.Register(ctx, topic, group)
	if err != nil {
		return fmt.Errorf("[event/consumer] failed to get consumer group for topic: %w", err)
	}

	c.subscriberWorkerPool = append(c.subscriberWorkerPool, SubscriberWorkerPool{
		workers:        c.workerPoolConfig.getWorkers(topic, group),
		consumerGroup:  consumerGroup,
		handler:        handler,
		topic:          topic,
		group:          group,
		commitStrategy: c.commitStrategy,
		newRelic:       c.newRelic,
	})

	return nil
}

func (c *Consumer) subscribeWithEventHandler(ctx context.Context, topic, group string, handler ConsumerHandler, errorHandlerLevel string, withBackOff bool) error {
	if c.isRunning() {
		return fmt.Errorf("[event/consumer] consumer already started, can't register new subscriber")
	}

	topic = c.eventConfig.getTopic(topic)

	consumerGroup, err := c.subscriber.Register(ctx, topic, group)
	if err != nil {
		return fmt.Errorf("[event/consumer] failed to get consumer group for topic: %w", err)
	}

	if c.eventHandler != nil {
		targetHandler := handler
		handler = func(ctx context.Context, message *EventConsumeMessage) error {
			log.WithFields(log.Fields{
				"topic": message.Topic,
				"key":   message.Key,
				"data":  string(message.Data),
			}).InfoWithCtx(ctx, "[event/consumer] Incoming message")

			err := c.eventHandler.HandleConsume(ctx, message, errorHandlerLevel, targetHandler, withBackOff)
			if err != nil {
				log.WithFields(log.Fields{
					"topic":    message.Topic,
					"key":      message.Key,
					"metadata": message.Metadata,
					"data":     string(message.Data),
					"error":    err,
				}).ErrorWithCtx(ctx, "[event/consumer] Consumer handler got an error")
				return err
			}

			return nil
		}
	}

	c.subscriberWorkerPool = append(c.subscriberWorkerPool, SubscriberWorkerPool{
		workers:        c.workerPoolConfig.getWorkers(topic, group),
		consumerGroup:  consumerGroup,
		handler:        handler,
		topic:          topic,
		group:          group,
		commitStrategy: c.commitStrategy,
		newRelic:       c.newRelic,
	})

	return nil
}

// Start activate the consumer and start receiving event
func (c *Consumer) Start() error {
	if c.isRunning() {
		return fmt.Errorf("consumer already started")
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	// double lock check
	// in case it started at the same time on different thread
	if c.isRunning() {
		return ErrConsumerStarted
	}

	c.stopch = make(chan bool)
	c.shutdown = make(chan bool)
	wg := sync.WaitGroup{}

	for i := range c.subscriberWorkerPool {
		pool := c.subscriberWorkerPool[i]

		wg.Add(pool.workers)
		go pool.run(c.stopch, &wg)
	}

	go func() {
		wg.Wait()
		close(c.shutdown)
	}()

	atomic.StoreUint32(&c.running, start)

	return nil
}

func (c *Consumer) RunWithHandlerConfig(cfg map[TopicName]ConsumerHandlerConfig) {
	for topic, handler := range cfg {
		consumerHandler := handler
		if topic != "" && consumerHandler.ConsumerGroup != "" {
			err := c.subscribeWithEventHandler(context.Background(), topic.String(), consumerHandler.ConsumerGroup, consumerHandler.Handler, consumerHandler.ErrorHandlerLevel, consumerHandler.WithBackOff)
			if err != nil {
				log.WithFields(log.Fields{
					"error":          err,
					"topic":          topic,
					"consumer-group": consumerHandler.ConsumerGroup,
				}).Warn("[event/consumer] Failed to subscribe to topic")
			}
		}
	}

	if err := c.Start(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("[event/consumer] Failed to start consumer")
	}

	log.Info("[event/consumer] Consumer is up and running!")

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	<-signalCh

	log.Info("[event/consumer] Terminating consumer")

	if err := c.Stop(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("[event/consumer] Error on stopping consumer")
	}
}

// Stop gracefully stop the consumer waiting for all workers to complete
// before exiting
func (c *Consumer) Stop() error {
	return c.StopContext(context.Background())
}

// StopContext gracefully stop the consumer or until the context timeout
func (c *Consumer) StopContext(ctx context.Context) error {
	if !c.isRunning() {
		return nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	// double lock check
	// in case it stop at the same time on different thread
	if !c.isRunning() {
		return nil
	}

	log.Print("[event/consumer] stopping consumer")
	close(c.stopch)

	log.Print("[event/consumer] waiting for all workers to stop")

	var err error
	select {
	case <-ctx.Done():
		log.Error("[event/consumer] timeout waiting consumer to stop")
		err = ctx.Err()
	case <-c.shutdown:
	}

	c.stopch = nil
	c.shutdown = nil
	atomic.StoreUint32(&c.running, stop)

	log.Print("[event/consumer] stopped")
	return err
}

func (c WorkerPoolConfig) getWorkers(topic, group string) int {
	config, err := c.parseTopicConfig(topic)
	if err != nil {
		if !errors.Is(err, errConfigNotFound) {
			log.WithError(err).Warn("failed to parse topic config, fallback to default")
		}

		return c.getDefaultWorkers(topic, group)
	}

	if val, ok := config[group]; ok && val > 0 {
		return val
	}

	if val, ok := config[MetaDefault]; ok && val > 0 {
		return val
	}

	return c.getDefaultWorkers(topic, group)
}

func (c WorkerPoolConfig) parseTopicConfig(topic string) (map[string]int, error) {
	if val, ok := c[topic]; ok && val != nil {
		config := map[string]int{}

		if err := mapstructure.Decode(val, &config); err != nil {
			return map[string]int{}, fmt.Errorf("failed to decode config: %w", err)
		}

		return config, nil
	}

	return nil, errConfigNotFound
}

func (c WorkerPoolConfig) getDefaultWorkers(topic, group string) int {
	if val, ok := c[MetaDefault].(int); ok && val > 0 {
		return val
	}

	return DefaultConsumerWorkers
}

func (s *SubscriberWorkerPool) run(stop <-chan bool, wg *sync.WaitGroup) {
	if closer, ok := s.consumerGroup.(Closer); ok {
		defer func(closer Closer) {
			if err := closer.Close(); err != nil {
				log.WithError(err).
					Error("[consumer/workerpool] failed to close consumer group")
			}
		}(closer)
	}

	err := s.consumerGroup.Start(context.Background())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("[consumer/workerpool] failed to start consumer group")

		// remove all wg for workers
		wg.Add(-1 * s.workers)
		return
	}

	jobs := make(chan Job, s.workers)

	for i := 0; i < s.workers; i++ {
		go worker(jobs, stop, wg)
	}

	for {
		select {
		case <-stop:
			close(jobs)
			return
		default:
			jobs <- s.retrieveMessage
		}
	}
}

func (s *SubscriberWorkerPool) retrieveMessage(ctx context.Context) (err error) {
	var txn *newrelic.Transaction
	defer func() {
		if txn != nil {
			txn.End()
		}
	}()

	ctx = context.WithValue(ctx, libCtx.RequestIdKey, uuid.NewString())

	if s.newRelic != nil {
		txn = s.newRelic.StartTransaction(fmt.Sprintf("%s:%s", s.group, s.topic))
		ctx = newrelic.NewContext(ctx, txn)
		ctx = context.WithValue(ctx, tracing.NewRelicTransactionKey, txn)
	}

	sentryTxn := sentry.StartTransaction(ctx, fmt.Sprintf("%s:%s", s.group, s.topic))
	defer sentryTxn.Finish()

	ctx = sentryTxn.Context()

	message, err := s.consumerGroup.GetMessage(ctx)
	if message == nil {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to get next item: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			stackTrace := string(debug.Stack())

			log.WithFields(log.Fields{
				"handler":     s.handler,
				"stack-trace": stackTrace,
				"error":       fmt.Sprintf("%+v", err),
			}).ErrorWithCtx(ctx, "[consumer.panicHandler] panic have occurred")

			if errR, ok := r.(error); !ok {
				err = fmt.Errorf(`[consumer] error when executing handler %+v`, errR)
			}
		}

	}()

	err = s.commitStrategy(ctx, message, s.handler)
	if err != nil {
		return fmt.Errorf("failed to consume message topic [%s] group [%s]: %w", s.topic, s.group, err)
	}
	return nil
}

func worker(jobs <-chan Job, stop <-chan bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-stop:
			log.Print("[consumer/worker] stop processing job")
			return
		case j := <-jobs:

			ctx, cancel := context.WithCancel(context.Background())
			resultCh := make(chan error, 1)

			go func(result chan<- error) {
				result <- j(ctx)
			}(resultCh)

			select {
			case <-stop:
				log.Print("[consumer/worker] stopping, cancel context and wait job to complete")

				cancel()
				if err := <-resultCh; err != nil {
					logger := log.WithError(err)

					if errors.Is(err, context.Canceled) {
						logger.WarnWithCtx(ctx, "[consumer/worker] cancel job error")
					} else {
						logger.ErrorWithCtx(ctx, "[consumer/worker] unexpected error while canceling job")
					}
				}

				close(resultCh)
				return
			case err := <-resultCh:
				if err != nil {
					log.WithError(err).ErrorWithCtx(ctx, "[consumer/worker] failed to complete job")
				}

				cancel()
				close(resultCh)
			}
		}
	}
}

// NewEventConsumeMessage return event consume message from byte data
func NewEventConsumeMessage(v []byte) (*EventConsumeMessage, error) {
	if v == nil {
		return &EventConsumeMessage{}, nil
	}

	var readmsg eventConsumeMessageRead
	if err := json.Unmarshal(v, &readmsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return &EventConsumeMessage{
		Metadata: readmsg.Metadata,
		Data:     readmsg.Data,
	}, nil
}

func (t TopicName) String() string {
	return string(t)
}
