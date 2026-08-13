package version

import (
	"encoding/json"
	"testing"

	contractevents "hospital/contracts/events"
)

func TestBuildChangedEvent(t *testing.T) {
	accountID := "20000000-0000-0000-0000-000000000002"
	operationID := "20000000-0000-0000-0000-000000000020"
	action := "identity.doctor.promoted"
	event, changed, err := BuildChangedEvent(accountID, operationID, action, 7, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected an authorization event")
	}
	if event.AggregateID != accountID ||
		event.EventType != contractevents.EventTypeIdentityAuthorizationChangedV1 ||
		event.SchemaVersion != 1 {
		t.Fatalf("unexpected pending event: %#v", event)
	}

	var payload contractevents.IdentityAuthorizationChangedV1Payload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OperationID != operationID ||
		payload.Action != action ||
		payload.AuthorizationVersion != 8 {
		t.Fatalf("unexpected authorization payload: %#v", payload)
	}
}

func TestBuildChangedEventSkipsProfileOnlyChange(t *testing.T) {
	event, changed, err := BuildChangedEvent("account-1", "operation-1", "identity.doctor.updated", 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if changed || event.AggregateID != "" || event.EventType != "" ||
		event.SchemaVersion != 0 || len(event.Payload) != 0 {
		t.Fatalf("profile-only change created an authorization event: %#v", event)
	}
}

func TestBuildChangedEventRejectsVersionDecrease(t *testing.T) {
	_, changed, err := BuildChangedEvent("account-1", "operation-1", "identity.account.changed", 8, 7)
	if err == nil || changed {
		t.Fatalf("expected version decrease to be rejected: changed=%t err=%v", changed, err)
	}
}
