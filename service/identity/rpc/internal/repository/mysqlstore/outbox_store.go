package mysqlstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"hospital/service/identity/rpc/internal/messaging/outbox"
)

var _ outbox.Store = (*Store)(nil)

func (s *Store) ListPending(ctx context.Context, eventType string, limit int) ([]outbox.Record, error) {
	if eventType == "" {
		return nil, errors.New("outbox event type is required")
	}
	if limit <= 0 {
		return nil, errors.New("outbox list limit must be positive")
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT event_id, aggregate_id, event_type, schema_version, payload, occurred_at, attempts
FROM identity_outbox_events
WHERE published_at IS NULL
  AND event_type = ?
ORDER BY occurred_at, event_id
LIMIT ?`, eventType, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending outbox events: %w", err)
	}
	defer rows.Close()

	records := make([]outbox.Record, 0, limit)
	for rows.Next() {
		var record outbox.Record
		var payload []byte
		if err := rows.Scan(
			&record.EventID,
			&record.AggregateID,
			&record.EventType,
			&record.SchemaVersion,
			&payload,
			&record.OccurredAt,
			&record.Attempts,
		); err != nil {
			return nil, fmt.Errorf("scan pending outbox event: %w", err)
		}
		record.Payload = append(json.RawMessage(nil), payload...)
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending outbox events: %w", err)
	}
	return records, nil
}

func (s *Store) MarkPublished(ctx context.Context, eventID string) error {
	if eventID == "" {
		return errors.New("outbox event ID is required")
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE identity_outbox_events
SET published_at = CURRENT_TIMESTAMP(3),
    last_error = NULL
WHERE event_id = ?
  AND published_at IS NULL`, eventID); err != nil {
		return fmt.Errorf("mark outbox event published: %w", err)
	}
	return nil
}

func (s *Store) MarkFailed(ctx context.Context, eventID string, failure string) error {
	if eventID == "" {
		return errors.New("outbox event ID is required")
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE identity_outbox_events
SET attempts = attempts + 1,
    last_error = LEFT(?, 512)
WHERE event_id = ?
  AND published_at IS NULL`, failure, eventID); err != nil {
		return fmt.Errorf("mark outbox event failed: %w", err)
	}
	return nil
}
