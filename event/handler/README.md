# Event Handler

Supported driver:
- Kafka


What's inside:
- event error handler
- retry default handler (will need cron to activate)
- interface for retry (will need self implementation on your service)


## Usage

### Event Error Handler
Advance usage combination with event consumer

```go
func main() {
    publisher, err := event.NewPublisher(context.Background(), &event.PublisherConfig{
        DriverConfig: &event.DriverConfig{
            Type: "kafka",
            Config: map[string]interface{}{
                "brokers":    []string{"localhost:9092"},
                "idempotent": true,
            },
        },
        EventConfig: &event.EventConfig{
            EventMap: map[string]string{
                "test": "kafka-test-topic", // map event test into topic kakfa-test-topic
			},
            Metadata: map[string]map[string]interface{}{
                "test": { // add metdata on event test
                    "route": "kafka-test",
                },
            },
        },
    })
    if err != nil {
        log.Fatal("failed to connect kafka")
    }
    
    db := sql.New(sql.DBConfig{
        SlaveDSN:      "root:user123@tcp(localhost:3306)/test-db",
        MasterDSN:     "root:user123@tcp(localhost:3306)/test-db",
        RetryInterval: 1,
    }, sql.DriverMySQL)
    
    eventLogRepo := NewEventLogRepo(db)
    
    eventHandler := handler.New(&handler.Options{
        RetryConfig: handler.RetryConfig{
            MaxRetry:          3,
            RetryInitialDelay: 10 * time.Millisecond,
            MaxJitter:         100 * time.Millisecond,
            HandlerTimeout:    20 * time.Second,
            BackOffConfig: []time.Duration{
                10 * time.Second,
                30 * time.Second,
            },
            MaxBackoffAttempt: 3,
        },
        Publisher:   *publisher,
        DlqTopic:    "dlq-topic",
        LogRepo:     eventLogRepo,
        ServiceName: "test-service",
    })
    
    consumer, err := event.NewConsumer(context.Background(), &event.ConsumerConfig{
    Consumer: &event.DriverConfig{
        Type: "kafka",
        Config: map[string]interface{}{
            "brokers":               []string{"localhost:9092"},
            "kafka_cluster_version": "3.2.0",
            },
        },
        WorkerPoolConfig: &event.WorkerPoolConfig{
            "default": 1, //default worker pool for each subscriber
        },
        CommitStrategy: &event.DriverConfig{
            Type:   "commit_on_success", // available strategy commit_on_success and always_commit, default: commit_on_success
            Config: map[string]interface{}{},
        },
    })
    if err != nil {
        log.Fatal("failed to create consumer")
    }
    
    // defined specific handler function
    handlerFn := func(ctx context.Context, message *event.EventConsumeMessage) error {
        // proceed message here
        log.WithFields(log.Fields{
        "message": fmt.Sprintf("%+v", message),
        "data":    string(message.Data),
        }).Info("Incoming message consumed")
        return nil
    }
    
    err = consumer.Subscribe(context.Background(), "kafka-test-topic", "test-group-id", func(ctx context.Context, message *event.EventConsumeMessage) error {
        // using event handler to handle consume message
        return eventHandler.HandleConsume(ctx, message, handler.ErrorHandlerPhase1, handlerFn, false)
    })
    if err != nil {
        log.Fatal("failed to subscribe topic")
    }
    
    if err := consumer.Start(); err != nil {
        log.WithError(err).Fatalln("failed to start consumer")
    }
    
    log.Info("Kafka consumer is up and running...")
    
    signalCh := make(chan os.Signal, 1)
    signal.Notify(signalCh, os.Interrupt)
    
    <-signalCh
    
    if err := consumer.Stop(); err != nil {
    log.WithError(err).Errorln("error on stopping consumer")
    }
}
```

You can wrap your specific handler with eventHandler using `HandleConsume` function. All event error handler already wrapped with that handler.

### Retry Cron
If you use `ErrorHandlerPhase2` with backOff, then you need the retry cron to fetch the data and publish it again via publisher.

```go
func main() {
    publisher, err := event.NewPublisher(context.Background(), &event.PublisherConfig{
        DriverConfig: &event.DriverConfig{
            Type: "kafka",
            Config: map[string]interface{}{
                "brokers":    []string{"localhost:9092"},
                "idempotent": true,
            },
        },
        EventConfig: &event.EventConfig{
            EventMap: map[string]string{
                "test": "kafka-test-topic", // map event test into topic kakfa-test-topic
            },
            Metadata: map[string]map[string]interface{}{
                "test": { // add metdata on event test
                    "route": "kafka-test",
                },
            },
        },
    })
    if err != nil {
        log.Fatal("failed to connect kafka")
    }
    
    db := sql.New(sql.DBConfig{
        SlaveDSN:      "root:user123@tcp(localhost:3306)/test-db",
        MasterDSN:     "root:user123@tcp(localhost:3306)/test-db",
        RetryInterval: 1,
    }, sql.DriverMySQL)
    
    eventLogRepo := NewEventLogRepo(db)
    
    redLock := redlock.New(&redlock.Config{
        Addr:      "localhost:6379",
        Namespace: "lock-redis",
        DB:        0,
    })
    
    eventRetry := handler.NewEventRetry(&handler.RetryOptions{
        EventLogRepo: eventLogRepo,
        Publisher:    *publisher,
        RedLock:      redLock,
        RetryConfig: handler.EventRetryConfig{
            PoolSize:  10,
            DelayLoad: 1 * time.Minute,
            MaxRetry:  3,
        },
    })
    
    // running event retry
    eventRetry.Run()
}
```

You need to add this job as long-running job, so it can fetch by your config delay load.

### Retry Log Repository Interface
If you want to use this event handler, you need to implement `RetryEventLogRepository`. This example only left with TODO, implement it by yourself based on your database.

```go
type eventLogImpl struct {
	// can use any other database
	db *sql.Store
}

func NewEventLogRepo(db *sql.Store) handler.RetryEventLogRepository {
	return &eventLogImpl{db: db}
}

func (e *eventLogImpl) Insert(ctx context.Context, log handler.EventRetryLog) error {
	//TODO implement me
	panic("implement me")
}

func (e *eventLogImpl) FindById(ctx context.Context, id uuid.UUID) (*handler.EventRetryLog, error) {
	//TODO implement me
	panic("implement me")
}

func (e *eventLogImpl) FindAll(ctx context.Context, filter *handler.LogFilter) ([]handler.EventRetryLog, error) {
	//TODO implement me
	panic("implement me")
}

func (e *eventLogImpl) UpdateById(ctx context.Context, id uuid.UUID, log *handler.EventLogToUpdate) error {
	//TODO implement me
	panic("implement me")
}
```

You need to implement it by yourself, since you may use sql/nosql db based on that.