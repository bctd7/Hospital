package outbox

import (
	"encoding/json"
	"fmt"

	contractevents "hospital/contracts/events"
	"hospital/service/identity/rpc/internal/identityadmin"
)

// PendingEvent contains the stable event facts that must be persisted in the
// same MySQL transaction as the business change. The MySQL adapter assigns the
// event ID and occurrence time when it inserts the row.
type PendingEvent struct {
	AggregateID   string
	EventType     string
	SchemaVersion int
	Payload       json.RawMessage
}

// BuildIdentityAuthorizationChanged maps an Identity account change to the
// integration event consumed by the Redis authorization-version projection.
// A profile-only change does not create an authorization event.
func BuildIdentityAuthorizationChanged(change identityadmin.Change) (PendingEvent, bool, error) {
	beforeVersion := change.Before.AuthorizationVersion
	afterVersion := change.After.AuthorizationVersion

	if afterVersion == beforeVersion {
		return PendingEvent{}, false, nil
	}
	if afterVersion < beforeVersion {
		return PendingEvent{}, false, fmt.Errorf(
			"authorization version decreased from %d to %d",
			beforeVersion,
			afterVersion,
		)
	}

	payload, err := json.Marshal(contractevents.IdentityAuthorizationChangedV1Payload{
		OperationID:          change.OperationID,
		Action:               change.Action,
		AuthorizationVersion: afterVersion,
	})
	if err != nil {
		return PendingEvent{}, false, fmt.Errorf("marshal authorization changed payload: %w", err)
	}

	return PendingEvent{
		AggregateID:   change.TargetAccountID,
		EventType:     contractevents.EventTypeIdentityAuthorizationChangedV1,
		SchemaVersion: 1,
		Payload:       payload,
	}, true, nil
}
