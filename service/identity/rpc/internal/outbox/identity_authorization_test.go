package outbox

import (
	"encoding/json"
	"testing"

	contractevents "hospital/contracts/events"
	"hospital/service/identity/rpc/internal/identityadmin"
)

func TestBuildIdentityAuthorizationChanged(t *testing.T) {
	change := identityadmin.Change{
		OperationID:     "20000000-0000-0000-0000-000000000020",
		TargetAccountID: "20000000-0000-0000-0000-000000000002",
		Action:          identityadmin.ActionPromoteDoctor,
		Before:          identityadmin.Account{AuthorizationVersion: 7},
		After:           identityadmin.Account{AuthorizationVersion: 8},
	}

	event, changed, err := BuildIdentityAuthorizationChanged(change)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected an authorization event")
	}
	if event.AggregateID != change.TargetAccountID ||
		event.EventType != contractevents.EventTypeIdentityAuthorizationChangedV1 ||
		event.SchemaVersion != 1 {
		t.Fatalf("unexpected pending event: %#v", event)
	}

	var payload contractevents.IdentityAuthorizationChangedV1Payload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OperationID != change.OperationID ||
		payload.Action != change.Action ||
		payload.AuthorizationVersion != 8 {
		t.Fatalf("unexpected authorization payload: %#v", payload)
	}
}

func TestBuildIdentityAuthorizationChangedSkipsProfileOnlyChange(t *testing.T) {
	event, changed, err := BuildIdentityAuthorizationChanged(identityadmin.Change{
		Action: identityadmin.ActionUpdateDoctor,
		Before: identityadmin.Account{AuthorizationVersion: 8},
		After:  identityadmin.Account{AuthorizationVersion: 8},
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed || event.AggregateID != "" || event.EventType != "" ||
		event.SchemaVersion != 0 || len(event.Payload) != 0 {
		t.Fatalf("profile-only change created an authorization event: %#v", event)
	}
}

func TestBuildIdentityAuthorizationChangedRejectsVersionDecrease(t *testing.T) {
	_, changed, err := BuildIdentityAuthorizationChanged(identityadmin.Change{
		Before: identityadmin.Account{AuthorizationVersion: 8},
		After:  identityadmin.Account{AuthorizationVersion: 7},
	})
	if err == nil || changed {
		t.Fatalf("expected version decrease to be rejected: changed=%t err=%v", changed, err)
	}
}
