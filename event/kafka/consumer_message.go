package kafka

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"golib/event"
)

type (
	ConsumeMessage struct {
		session sarama.ConsumerGroupSession
		message *sarama.ConsumerMessage
	}
)

func (cm *ConsumeMessage) GetEventConsumeMessage(ctx context.Context) (*event.EventConsumeMessage, error) {
	ecm, err := event.NewEventConsumeMessage(cm.message.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}
	ecm.Topic = cm.message.Topic

	if len(cm.message.Key) > 0 {
		ecm.Key = string(cm.message.Key)
	}

	return ecm, nil
}

func (cm *ConsumeMessage) Commit(ctx context.Context) error {
	cm.session.MarkMessage(cm.message, "")
	return nil
}
