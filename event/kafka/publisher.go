package kafka

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/mitchellh/mapstructure"
	"golib/event"
	"golib/log"
	"golib/tracing"
	"strconv"
	"time"
)

type (
	Publisher struct {
		producer sarama.SyncProducer
	}
	PublisherConfig struct {
		Brokers []string `json:"brokers" mapstructure:"brokers"`

		// The maximum permitted size of a message (defaults to 1000000). Should be
		// set equal to or smaller than the broker's `message.max.bytes`.
		MaxMessageBytes int `json:"max_message_bytes" mapstructure:"max_message_bytes"`

		// The level of acknowledgement reliability needed from the broker (defaults
		// to WaitForLocal). Equivalent to the `request.required.acks` setting of the
		// JVM producer.
		Acks string `json:"acks" mapstructure:"acks"`

		// The maximum duration the broker will wait the receipt of the number of
		// RequiredAcks (defaults to 10 seconds). This is only relevant when
		// RequiredAcks is set to WaitForAll or a number > 1. Only supports
		// millisecond resolution, nanoseconds will be truncated. Equivalent to
		// the JVM producer's `request.timeout.ms` setting.
		Timeout string `json:"timeout" mapstructure:"timeout"`

		// TLS Config
		KeyFile       string `json:"key_file" mapstructure:"key_file"`
		CertFile      string `json:"cert_file" mapstructure:"cert_file"`
		CACertificate string `json:"ca_cert" mapstructure:"ca_cert"`

		// If enabled, the producer will ensure that exactly one copy of each message is written.
		Idempotent bool `json:"idempotent" mapstructure:"idempotent"`

		// Producer attempt to publish message
		MaxRetry int `json:"max_retry" mapstructure:"max_retry"`
	}
)

func NewKafkaPublisher(_ context.Context, config any) (event.Emitter, error) {
	var publisherCfg PublisherConfig
	if err := mapstructure.Decode(config, &publisherCfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	saramaCfg := sarama.NewConfig()

	if publisherCfg.Acks != "" {
		ack, err := strconv.Atoi(publisherCfg.Acks)
		if err != nil {
			return nil, fmt.Errorf("invalid kafka producer acks value: %w", err)
		}
		saramaCfg.Producer.RequiredAcks = sarama.RequiredAcks(ack)
	} else {
		saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	}

	if publisherCfg.MaxMessageBytes > 0 {
		saramaCfg.Producer.MaxMessageBytes = publisherCfg.MaxMessageBytes
	}

	if publisherCfg.Timeout != "" {
		timeout, err := time.ParseDuration(publisherCfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid kafka producer timeout value: %w", err)
		}
		saramaCfg.Producer.Timeout = timeout
	}

	saramaCfg.Producer.Return.Successes = true
	saramaCfg.Producer.Idempotent = publisherCfg.Idempotent
	if saramaCfg.Producer.Idempotent {
		saramaCfg.Net.MaxOpenRequests = 1
	}

	if publisherCfg.MaxRetry > 0 {
		saramaCfg.Producer.Retry.Max = publisherCfg.MaxRetry
	}

	tls := createTlsConfig(publisherCfg.CertFile, publisherCfg.KeyFile, publisherCfg.CACertificate)
	if tls != nil {
		saramaCfg.Net.TLS.Config = tls
		saramaCfg.Net.TLS.Enable = true
	}

	producer, err := sarama.NewSyncProducer(publisherCfg.Brokers, saramaCfg)
	if err != nil {
		log.Fatal("Failed to start kafka producer:", err)
	}

	return &Publisher{producer: producer}, nil
}

func (p *Publisher) Publish(ctx context.Context, message *event.Message) error {
	span, ctx := tracing.StartSpanFromContext(ctx, "KafkaPublisher.Publish")
	defer span.End()

	mb, err := message.ToBytes()
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: message.Topic,
		Value: sarama.StringEncoder(mb),
	}

	if message.Key != "" {
		msg.Key = sarama.StringEncoder(message.Key)
	}

	_, _, err = p.producer.SendMessage(msg)
	if err != nil {
		return err
	}

	return nil
}
