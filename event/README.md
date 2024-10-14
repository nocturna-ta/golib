# Event

## Event Publisher

Supported driver:
- Kafka

### Usage

Don't forget to add this line `_ "github.com/scprimesolution/golib/event/kafka"` because it will trigger init function. If not import then the code will not work.

```go
import (
	"context"
	"github.com/scprimesolution/golib/event"
	_ "github.com/scprimesolution/golib/event/kafka"
	"github.com/scprimesolution/golib/log"
)

type Message struct {
	Name string `json:"name"`
}

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

	// publish message
	err = publisher.Publish(context.Background(), "test", "", Message{Name: "test"}, map[string]interface{}{})
	if err != nil {
		log.Fatal("failed to publish message")
	}
}
```

### Driver Config

Defined configuration for driver. See supported config at [PublisherConfig](kafka/publisher.go)

### Event Config

Sometimes, kafka topic maybe too long or you need to add default metadata on kafka but you don't want to add it everytime, so you need to add EventConfig for that.

### API

```go
Publish(ctx context.Context, event, key string, message interface{}, metadata map[string]interface{}) error
```

- event -> your event name, it can be kafka topic
- key -> on kafka you can guarantee it published to same partition if you defined kafka key, the same kafka key will publish to same partition
- message -> your message that need to publish
- metadata -> additional information



## Event Consumer

Supported driver:
- Kafka

### Usage

Don't forget to add this line `_ "github.com/scprimesolution/golib/event/kafka"` because it will trigger init function. If not import then the code will not work.

```go
import (
	"context"
	"fmt"
	"github.com/scprimesolution/golib/event"
	_ "github.com/scprimesolution/golib/event/kafka"
	"github.com/scprimesolution/golib/log"
	"os"
	"os/signal"
)

func main() {
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

	err = consumer.Subscribe(context.Background(), "kafka-test-topic", "test-group-id", func(ctx context.Context, message *event.EventConsumeMessage) error {
		// proceed message here
		log.WithFields(log.Fields{
			"message": fmt.Sprintf("%+v", message),
			"data":    string(message.Data),
		}).Info("Incoming message consumed")
		return nil
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