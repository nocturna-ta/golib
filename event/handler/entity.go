package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

type LogStatus string

const (
	WaitingRetry             LogStatus = "WAITING_RETRY"
	Failed                   LogStatus = "FAILED"
	SortDirAsc                         = "ASC"
	SortDirDesc                        = "DESC"
	DefaultMaxBackoffAttempt           = 3
)

type (
	EventRetryLog struct {
		ID          uuid.UUID       `db:"id" bson:"_id,omitempty" `
		SourceTopic string          `db:"source_topic" bson:"source_topic"`
		ServiceName string          `db:"service_name" bson:"service_name"`
		Data        json.RawMessage `db:"data" bson:"data"`
		Status      LogStatus       `db:"status" bson:"status"`
		Count       int             `db:"count" bson:"count"`
		BackOffDate time.Time       `db:"backoff_date" bson:"backoff_date"`
		Error       *string         `db:"error" bson:"error,omitempty"`
		CreatedDate time.Time       `db:"created_time" bson:"created_date"`
		UpdatedDate *time.Time      `db:"updated_time" bson:"updated_date,omitempty"`
	}

	DlqMessage struct {
		Data        any            `json:"data"`
		SourceTopic string         `json:"source_topic"`
		Metadata    map[string]any `json:"metadata"`
		ServiceName string         `json:"service_name"`
		Timestamp   time.Time      `json:"timestamp"`
		Error       string         `json:"error"`
	}

	BackoffRetry struct {
		SourceTopic string         `json:"source_topic"`
		Data        []byte         `json:"data"`
		Metadata    map[string]any `json:"metadata"`
		Timestamp   time.Time      `json:"timestamp"`
		Error       string         `json:"error"`
	}

	DefaultFilter struct {
		SortBy  string
		SortDir string
		Page    int
		Limit   int
	}

	LogFilter struct {
		DefaultFilter
		MaxRetry       int
		MaxBackOffDate *time.Time
		InStatus       []LogStatus
	}

	EventLogToUpdate struct {
		Status      *LogStatus
		Count       *int
		BackOffDate *time.Time
		Error       *string
	}
)

func (df *DefaultFilter) GetSkip() int {
	return (df.Page - 1) * df.Limit
}

func (df *DefaultFilter) GetLimit() int {
	return df.Limit
}

func (df *DefaultFilter) GetSortDir() string {
	isDescending := bytes.EqualFold([]byte(df.SortDir), []byte(SortDirDesc))
	if isDescending {
		return SortDirDesc
	}
	return SortDirAsc
}

func (df *DefaultFilter) GetSortDirInt64() int64 {
	isDescending := bytes.EqualFold([]byte(df.SortDir), []byte(SortDirDesc))
	if isDescending {
		return -1
	}
	return 1
}

func (df *DefaultFilter) Validate() {
	if df.Page <= 0 {
		df.Page = 1
	}
	if df.Limit <= 0 || df.Limit > 1000 {
		df.Limit = 10
	}
}

func (f *LogFilter) GetSortBy() string {
	switch f.SortBy {
	case "id":
		return "id"
	case "_id":
		return "_id"
	case "backoff_date":
		return "backoff_date"
	default:
		return "id"
	}
}

func GetBackOffTopic(topic string, attempts int) string {
	return fmt.Sprintf("%s-backoff-attempts-%d", getBaseTopic(topic), attempts)
}

func IsBackOffTopic(topic string) bool {
	return strings.Contains(topic, "-backoff-attempts-")
}

func getBaseTopic(topic string) string {
	split := strings.Split(topic, "-backoff-attempts-")
	return split[0]
}

func (k *EventRetryLog) IsAllowRetry(maxRetry int) bool {
	maxRetryLog := DefaultMaxBackoffAttempt
	if maxRetry > 0 {
		maxRetryLog = maxRetry
	}
	isFailed := k.Status == Failed
	validCount := k.Count <= maxRetryLog
	return isFailed && validCount
}
