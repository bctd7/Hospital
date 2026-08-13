// Package version owns authorization-version integration events and the
// Kafka-to-Redis consumer that makes changed versions available to interceptors.
package version

import (
	"encoding/json"
	"fmt"

	contractevents "hospital/contracts/events"
)

// Event contains the stable authorization facts persisted through the Outbox.
// The MySQL adapter assigns the event ID and occurrence time.
type Event struct {
	AggregateID   string
	EventType     string
	SchemaVersion int
	Payload       json.RawMessage
}

// BuildChangedEvent creates the authorization integration event owned by this
// domain. A profile-only account change does not produce an event.
func BuildChangedEvent(accountID, operationID, action string, beforeVersion, afterVersion int64) (Event, bool, error) {
	if afterVersion == beforeVersion {
		return Event{}, false, nil
	}
	if afterVersion < beforeVersion {
		return Event{}, false, fmt.Errorf(
			"authorization version decreased from %d to %d",
			beforeVersion,
			afterVersion,
		)
	}

	payload, err := json.Marshal(contractevents.IdentityAuthorizationChangedV1Payload{
		OperationID:          operationID,
		Action:               action,
		AuthorizationVersion: afterVersion,
	})
	if err != nil {
		return Event{}, false, fmt.Errorf("marshal authorization changed payload: %w", err)
	}

	return Event{
		AggregateID:   accountID,
		EventType:     contractevents.EventTypeIdentityAuthorizationChangedV1,
		SchemaVersion: 1,
		Payload:       payload,
	}, true, nil
}
