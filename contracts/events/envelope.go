package events

import (
	"encoding/json"
	"time"
)

type Envelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Producer      string          `json:"producer"`
	TraceID       string          `json:"trace_id,omitempty"`
	AggregateID   string          `json:"aggregate_id"`
	SchemaVersion int             `json:"schema_version"`
	Payload       json.RawMessage `json:"payload"`
}
