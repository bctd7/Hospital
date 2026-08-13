// Package outbox reliably publishes already-persisted integration events.
// Event meaning and construction belong to their owning domain packages.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	contractevents "hospital/contracts/events"
)

// Producer is the message publishing capability required by the Outbox
// publisher. The Kafka adapter satisfies this interface.
type Producer interface {
	Publish(ctx context.Context, topic, key string, message []byte) error
}

// Publisher moves persisted Outbox events from MySQL to a message topic.
type Publisher struct {
	store        Store
	producer     Producer
	topic        string
	eventType    string
	producerName string
	batchSize    int
}

func NewPublisher(
	store Store,
	producer Producer,
	topic string,
	eventType string,
	producerName string,
	batchSize int,
) (*Publisher, error) {
	if store == nil {
		return nil, errors.New("outbox store is required")
	}
	if producer == nil {
		return nil, errors.New("outbox producer is required")
	}
	if topic == "" {
		return nil, errors.New("outbox topic is required")
	}
	if eventType == "" {
		return nil, errors.New("outbox event type is required")
	}
	if producerName == "" {
		return nil, errors.New("outbox producer name is required")
	}
	if batchSize <= 0 {
		return nil, errors.New("outbox batch size must be positive")
	}

	return &Publisher{
		store:        store,
		producer:     producer,
		topic:        topic,
		eventType:    eventType,
		producerName: producerName,
		batchSize:    batchSize,
	}, nil
}

// PublishBatch publishes at most one configured batch of pending events. Each
// event is marked independently, so one failed event does not hide successful
// publication of the remaining events.
func (p *Publisher) PublishBatch(ctx context.Context) error {
	records, err := p.store.ListPending(ctx, p.eventType, p.batchSize)
	if err != nil {
		return fmt.Errorf("list pending outbox batch: %w", err)
	}

	var batchErr error
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return errors.Join(batchErr, err)
		}
		if err := p.publishRecord(ctx, record); err != nil {
			batchErr = errors.Join(batchErr, err)
		}
	}
	return batchErr
}

func (p *Publisher) publishRecord(ctx context.Context, record Record) error {
	message, err := json.Marshal(contractevents.Envelope{
		EventID:       record.EventID,
		EventType:     record.EventType,
		OccurredAt:    record.OccurredAt,
		Producer:      p.producerName,
		AggregateID:   record.AggregateID,
		SchemaVersion: record.SchemaVersion,
		Payload:       record.Payload,
	})
	if err != nil {
		return p.markFailed(ctx, record.EventID, fmt.Errorf("marshal outbox event envelope: %w", err))
	}

	if err := p.producer.Publish(ctx, p.topic, record.AggregateID, message); err != nil {
		return p.markFailed(ctx, record.EventID, err)
	}
	if err := p.store.MarkPublished(ctx, record.EventID); err != nil {
		return fmt.Errorf("mark outbox event %s published after Kafka ACK: %w", record.EventID, err)
	}
	return nil
}

func (p *Publisher) markFailed(ctx context.Context, eventID string, cause error) error {
	if err := p.store.MarkFailed(ctx, eventID, cause.Error()); err != nil {
		return errors.Join(cause, fmt.Errorf("record outbox event %s failure: %w", eventID, err))
	}
	return cause
}
