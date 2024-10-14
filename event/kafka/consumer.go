package kafka

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/mitchellh/mapstructure"
	"github.com/nocturna-ta/golib/event"
	"log"
	"strconv"
)

type (
	Consumer struct {
		brokers      []string
		saramaConfig *sarama.Config
	}

	ConsumerConfig struct {
		Brokers []string `json:"brokers" mapstructure:"brokers"`

		// Consumer group ID
		ConsumerGroup string `json:"consumer_group" mapstructure:"consumer_group"`

		// Kafka Cluster Version
		KafkaClusterVersion string `json:"kafka_cluster_version" mapstructure:"kafka_cluster_version"`

		// TLS Config
		KeyFile       string `json:"key_file" mapstructure:"key_file"`
		CertFile      string `json:"cert_file" mapstructure:"cert_file"`
		CACertificate string `json:"ca_cert" mapstructure:"ca_cert"`

		RebalanceStrategy string `json:"rebalance_strategy" mapstructure:"rebalance_strategy"`
		IsolationLevel    string `json:"isolation_level" mapstructure:"isolation_level"`
	}
)

func NewKafkaConsumer(_ context.Context, config any) (event.Subscriber, error) {
	var consumerCfg ConsumerConfig
	if err := mapstructure.Decode(config, &consumerCfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	saramaCfg := sarama.NewConfig()
	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	if consumerCfg.KafkaClusterVersion == "" {
		return nil, fmt.Errorf("should define kafka cluster version")
	} else {
		version, err := sarama.ParseKafkaVersion(consumerCfg.KafkaClusterVersion)
		if err != nil {
			return nil, fmt.Errorf("failed parsing Kafka version: %v", err)
		}
		saramaCfg.Version = version
	}

	if consumerCfg.RebalanceStrategy != "" {
		switch consumerCfg.RebalanceStrategy {
		case "sticky":
			saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategySticky}
		case "roundrobin":
			saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRoundRobin}
		case "range":
			saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.BalanceStrategyRange}
		default:
			log.Fatalf("Unrecognized consumer group partition rebalance strategy: %s", consumerCfg.RebalanceStrategy)
		}
	}

	if consumerCfg.IsolationLevel != "" {
		isolation, err := strconv.Atoi(consumerCfg.IsolationLevel)
		if err != nil {
			return nil, err
		}
		saramaCfg.Consumer.IsolationLevel = sarama.IsolationLevel(isolation)
	}

	tls := createTlsConfig(consumerCfg.CertFile, consumerCfg.KeyFile, consumerCfg.CACertificate)
	if tls != nil {
		saramaCfg.Net.TLS.Config = tls
		saramaCfg.Net.TLS.Enable = true
	}

	_, err := sarama.NewClient(consumerCfg.Brokers, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &Consumer{
		brokers:      consumerCfg.Brokers,
		saramaConfig: saramaCfg,
	}, nil
}

func (c *Consumer) Register(ctx context.Context, topic string, group string) (event.ConsumerGroup, error) {
	client, err := sarama.NewClient(c.brokers, c.saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	consumerGroup, err := sarama.NewConsumerGroupFromClient(group, client)
	if err != nil {
		return nil, fmt.Errorf("failed to register kafka consumer group: %w", err)
	}

	return &ConsumerGroup{
		consumerGroup: consumerGroup,
		topic:         topic,
		ready:         make(chan bool),
		message:       make(chan ConsumeMessage),
	}, nil
}
