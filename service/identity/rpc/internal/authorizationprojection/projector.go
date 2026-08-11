package authorizationprojection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"hospital/common/authn"
	contractevents "hospital/contracts/events"
)

// Projector applies authorization-change events to the Redis version
// projection. It does not fetch Kafka records or commit Kafka offsets.
type Projector struct {
	versions authn.AuthorizationVersionAdvancer
}

func NewProjector(versions authn.AuthorizationVersionAdvancer) (*Projector, error) {
	if versions == nil {
		return nil, errors.New("authorization version advancer is required")
	}
	return &Projector{versions: versions}, nil
}

// Apply validates one Kafka message and advances the corresponding Redis
// authorization version. updated=false with a nil error means that the event
// was a duplicate or older than the version already stored in Redis.
func (p *Projector) Apply(
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

	updated, err = p.versions.AdvanceAuthorizationVersion(
		ctx,
		accountID,
		payload.AuthorizationVersion,
	)
	if err != nil {
		return false, fmt.Errorf("project authorization version: %w", err)
	}
	return updated, nil
}
