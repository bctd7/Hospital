package outbox

import (
	"context"
	"time"
)

// Record represents an Outbox event that has already been persisted in MySQL.
type Record struct {
	PendingEvent

	EventID    string
	OccurredAt time.Time
	Attempts   int
}

// Store defines the persistence operations required by the Outbox publisher.
type Store interface {
	ListPending(ctx context.Context, eventType string, limit int) ([]Record, error)
	MarkPublished(ctx context.Context, eventID string) error
	MarkFailed(ctx context.Context, eventID string, failure string) error
}
