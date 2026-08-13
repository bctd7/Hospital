package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	contractevents "hospital/contracts/events"
)

// Consumer owns the complete Kafka -> Redis -> offset-commit workflow for
// authorization-version events.
type Consumer struct {
	messages MessageReader
	versions Store
}

func NewConsumer(messages MessageReader, versions Store) (*Consumer, error) {
	if messages == nil || versions == nil {
		return nil, errors.New("authorization version consumer dependencies are required")
	}
	return &Consumer{messages: messages, versions: versions}, nil
}

// Run retries a fetched record until Redis is updated, then commits its Kafka
// offset. A duplicate delivery is safe because version advances are monotonic.
func (c *Consumer) Run(ctx context.Context, retryInterval time.Duration) {
	if retryInterval <= 0 {
		retryInterval = time.Second
	}
	for {
		record, err := c.messages.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logx.Errorf("fetch identity authorization event: %v", err)
			if !waitForRetry(ctx, retryInterval) {
				return
			}
			continue
		}

		for {
			_, err = c.apply(ctx, string(record.Key), record.Value)
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				return
			}
			logx.Errorf("apply identity authorization event topic=%s partition=%d offset=%d: %v", record.Topic, record.Partition, record.Offset, err)
			if !waitForRetry(ctx, retryInterval) {
				return
			}
		}

		for {
			if err = c.messages.CommitMessage(ctx, record); err == nil {
				break
			}
			if ctx.Err() != nil {
				return
			}
			logx.Errorf("commit identity authorization event topic=%s partition=%d offset=%d: %v", record.Topic, record.Partition, record.Offset, err)
			if !waitForRetry(ctx, retryInterval) {
				return
			}
		}
	}
}

func waitForRetry(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (c *Consumer) apply(
	ctx context.Context,
	messageKey string,
	message []byte,
) (updated bool, err error) {
	if len(message) == 0 {
		return false, errors.New("authorization event message is required")
	}

	var envelope contractevents.Envelope
	if err := json.Unmarshal(message, &envelope); err != nil {
		return false, fmt.Errorf("decode authorization event envelope: %w", err)
	}
	if envelope.EventType != contractevents.EventTypeIdentityAuthorizationChangedV1 {
		return false, fmt.Errorf("unsupported authorization event type %q", envelope.EventType)
	}
	if envelope.SchemaVersion != 1 {
		return false, fmt.Errorf("unsupported authorization event schema version %d", envelope.SchemaVersion)
	}
	accountID := strings.TrimSpace(envelope.AggregateID)
	if accountID == "" {
		return false, errors.New("authorization event aggregate_id is required")
	}
	messageKey = strings.TrimSpace(messageKey)
	if messageKey == "" {
		return false, errors.New("authorization event Kafka key is required")
	}
	if messageKey != accountID {
		return false, errors.New("authorization event Kafka key must match aggregate_id")
	}

	var payload contractevents.IdentityAuthorizationChangedV1Payload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return false, fmt.Errorf("decode authorization event payload: %w", err)
	}
	if strings.TrimSpace(payload.OperationID) == "" {
		return false, errors.New("authorization event operation_id is required")
	}
	if strings.TrimSpace(payload.Action) == "" {
		return false, errors.New("authorization event action is required")
	}
	if payload.AuthorizationVersion <= 0 {
		return false, errors.New("authorization event version must be positive")
	}

	updated, err = c.versions.AdvanceAuthorizationVersion(
		ctx,
		accountID,
		payload.AuthorizationVersion,
	)
	if err != nil {
		return false, fmt.Errorf("advance authorization version: %w", err)
	}
	return updated, nil
}
