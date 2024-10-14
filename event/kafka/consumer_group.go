package kafka

import (
	"context"
	"github.com/IBM/sarama"
	"golib/event"
	"golib/log"
)

type (
	ConsumerGroup struct {
		consumerGroup sarama.ConsumerGroup
		topic         string
		ready         chan bool
		message       chan ConsumeMessage
	}
)

func (g *ConsumerGroup) Close() error {
	return g.consumerGroup.Close()
}

func (g *ConsumerGroup) Start(ctx context.Context) error {
	go func() {
		for {
			if err := g.consumerGroup.Consume(ctx, []string{g.topic}, g); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Errorf("Error from consumer: %v", err)
			}

			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}

			g.ready = make(chan bool)
		}
	}()

	<-g.ready
	return nil
}

func (g *ConsumerGroup) GetMessage(ctx context.Context) (event.ConsumeMessage, error) {
	select {
	case msg := <-g.message:
		return &msg, nil
	case <-ctx.Done():
		return nil, nil
	}
}

func (g *ConsumerGroup) Setup(sarama.ConsumerGroupSession) error {
	close(g.ready)
	return nil
}

func (g *ConsumerGroup) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (g *ConsumerGroup) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			g.message <- ConsumeMessage{
				session: session,
				message: message,
			}
		case <-session.Context().Done():
			return nil
		}
	}
}
