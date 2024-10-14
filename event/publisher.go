package event

import (
	"context"
	"errors"
	libCtx "github.com/nocturna-ta/golib/context"
	"time"
)

var (
	publishers = map[string]EmitterFactory{}
)

type (
	Publisher struct {
		emitter     Emitter
		eventConfig *EventConfig
	}

	PublisherConfig struct {
		DriverConfig *DriverConfig `json:"driver_config" mapstructure:"driver_config"`
		EventConfig  *EventConfig  `json:"event_config" mapstructure:"event_config"`
	}

	EmitterFactory func(ctx context.Context, config any) (Emitter, error)
)

type (
	Emitter interface {
		Publish(ctx context.Context, message *Message) error
	}

	MessagePublisher interface {
		Publish(ctx context.Context, event, key string, message any, metadata map[string]any) error
	}
)

func RegisterPublisher(name string, factory EmitterFactory) {
	publishers[name] = factory
}

func NewPublisher(ctx context.Context, config *PublisherConfig) (*Publisher, error) {
	if config == nil {
		return nil, errors.New("[event/publisher] missing config")
	}

	pub := &Publisher{
		eventConfig: config.EventConfig,
	}

	if config.DriverConfig == nil {
		return nil, errors.New("[event/publisher] no driver config")
	}

	if pub.eventConfig == nil {
		pub.eventConfig = NewEventConfig()
	}

	em, ok := publishers[config.DriverConfig.Type]
	if !ok {
		return nil, errors.New("[event/publisher] unsupported publisher driver")
	}

	emitter, err := em(ctx, config.DriverConfig.Config)
	if err != nil {
		return nil, err
	}

	pub.emitter = emitter

	return pub, nil
}

func (p *Publisher) Publish(ctx context.Context, event, key string, message any, metadata map[string]any) (err error) {
	if p.emitter == nil {
		return errors.New("[event/publisher] driver is not set")
	}

	topic := p.eventConfig.getTopic(event)
	md := p.eventConfig.getMetadata(event)
	if metadata == nil {
		metadata = map[string]any{}
	}

	for k, v := range md {
		metadata[k] = v
	}

	metadata, err = p.addRequestContextMetadata(ctx, metadata)
	if err != nil {
		return err
	}

	mHash, err := hash(message)
	if err != nil {
		return err
	}

	metadata[MetaHash] = mHash
	metadata[MetaTime] = time.Now()

	msg := &Message{
		Topic:    topic,
		Key:      key,
		Data:     message,
		Metadata: metadata,
	}

	return p.emitter.Publish(ctx, msg)
}

func (p *Publisher) addRequestContextMetadata(ctx context.Context, metadata map[string]any) (map[string]any, error) {
	reqCtx, err := libCtx.GetRequestContext(ctx)
	if err != nil {
		return nil, err
	}

	if reqCtx.RequestId != "" {
		metadata[libCtx.XRequestId] = reqCtx.RequestId
	}

	if reqCtx.UserId != "" {
		metadata[libCtx.XUserId] = reqCtx.UserId
	}

	if reqCtx.ChannelId != "" {
		metadata[libCtx.XChannelId] = reqCtx.ChannelId
	}

	if reqCtx.AccountId != "" {
		metadata[libCtx.XAccountId] = reqCtx.AccountId
	}

	return metadata, nil
}
