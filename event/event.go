package event

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	libCtx "github.com/nocturna-ta/golib/context"
)

const (
	MetaHash    = "hash"
	MetaTime    = "timestamp"
	MetaEvent   = "event"
	MetaVersion = "version"
	MetaDefault = "default"
)

var (
	errConfigNotFound  = errors.New("config not found")
	ErrConsumerStarted = errors.New("consumer already started")
)

type (
	Message struct {
		Topic    string         `json:"-"`
		Key      string         `json:"-"`
		Data     any            `json:"data,omitempty" mapstructure:"data"`
		Metadata map[string]any `json:"metadata,omitempty" mapstructure:"metadata"`
		RawData  []byte         `json:"-"` //To provide raw data to consumer
	}

	DriverConfig struct {
		Type   string `json:"type" mapstructure:"type"`
		Config any    `json:"config" mapstructure:"config"`
	}

	EventConfig struct {
		Metadata map[string]map[string]any `json:"metadata,omitempty" mapstructure:"metadata"`
		EventMap map[string]string         `json:"event_map,omitempty" mapstructure:"event_map"`
	}

	ConsumerHandler func(ctx context.Context, message *EventConsumeMessage) error
)

func NewEventConfig() *EventConfig {
	return &EventConfig{
		Metadata: make(map[string]map[string]any),
		EventMap: make(map[string]string),
	}
}

func (m *Message) ToBytes() ([]byte, error) {
	return json.Marshal(m)
}

func (c *EventConfig) getTopic(event string) string {
	if t, ok := c.EventMap[event]; ok {
		return t
	}
	return event
}

func (c *EventConfig) getMetadata(event string) map[string]any {
	if m, ok := c.getMetadataCopy(event); ok {
		m[MetaEvent] = event
		return m
	}
	return c.getDefaultMetadata(event)
}

func (c *EventConfig) getDefaultMetadata(event string) map[string]any {
	if m, ok := c.getMetadataCopy(MetaDefault); ok {
		m[MetaEvent] = event
		return m
	}

	return map[string]any{
		MetaVersion: 1,
		MetaEvent:   event,
	}
}

func (c *EventConfig) getMetadataCopy(name string) (map[string]any, bool) {
	if m, ok := c.Metadata[name]; ok {
		copyMap := map[string]any{}
		for k, v := range m {
			copyMap[k] = v
		}
		return copyMap, true
	}
	return nil, false
}

func hash(m any) (string, error) {
	mb, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	k := sha256.Sum256(mb)

	return string(base64.StdEncoding.EncodeToString(k[:])), nil
}

func requestContextFromMetadata(ctx context.Context, message *EventConsumeMessage) context.Context {
	var requestId, userId, channelId, addressId string
	if val, ok := message.Metadata[libCtx.XRequestId]; ok {
		if valStr, ok := val.(string); ok {
			requestId = valStr
		}
	}

	if val, ok := message.Metadata[libCtx.XRequestId]; ok {
		if valStr, ok := val.(string); ok {
			requestId = valStr
		} else {
			requestId = libCtx.ReadRequestId(ctx)
		}
	} else {
		requestId = libCtx.ReadRequestId(ctx)
		if requestId == "" {
			requestId = uuid.New().String()
		}
	}

	if val, ok := message.Metadata[libCtx.XUserId]; ok {
		if valStr, ok := val.(string); ok {
			userId = valStr
		}
	}

	if val, ok := message.Metadata[libCtx.XChannelId]; ok {
		if valStr, ok := val.(string); ok {
			channelId = valStr
		}
	}

	if val, ok := message.Metadata[libCtx.XAddressId]; ok {
		if valStr, ok := val.(string); ok {
			addressId = valStr
		}

	}

	reqCtx := libCtx.RequestContext{
		UserId:    userId,
		RequestId: requestId,
		ChannelId: channelId,
		Address:   addressId,
	}

	ctx = context.WithValue(ctx, libCtx.RequestIdKey, requestId)
	ctx = context.WithValue(ctx, libCtx.RequestContextKey, reqCtx)

	return ctx
}
