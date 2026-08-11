package authorizationprojection

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	contractevents "hospital/contracts/events"
)

func TestProjectorAppliesAuthorizationVersionEvent(t *testing.T) {
	advancer := &fakeAdvancer{updated: true}
	projector, err := NewProjector(advancer)
	if err != nil {
		t.Fatal(err)
	}

	message := authorizationEventMessage(t, "account-1", 7)
	updated, err := projector.Apply(context.Background(), "account-1", message)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("expected authorization version to be updated")
	}
	if advancer.accountID != "account-1" || advancer.version != 7 {
		t.Fatalf("unexpected projection call: account=%q version=%d", advancer.accountID, advancer.version)
	}
}

func TestProjectorRejectsMismatchedKafkaKey(t *testing.T) {
	advancer := &fakeAdvancer{}
	projector, err := NewProjector(advancer)
	if err != nil {
		t.Fatal(err)
	}

	_, err = projector.Apply(
		context.Background(),
		"different-account",
		authorizationEventMessage(t, "account-1", 7),
	)
	if err == nil {
		t.Fatal("expected mismatched Kafka key to be rejected")
	}
	if advancer.called {
		t.Fatal("authorization version must not be advanced for an invalid event")
	}
}

func TestProjectorReturnsRedisFailure(t *testing.T) {
	wantErr := errors.New("redis unavailable")
	projector, err := NewProjector(&fakeAdvancer{err: wantErr})
	if err != nil {
		t.Fatal(err)
	}

	_, err = projector.Apply(
		context.Background(),
		"account-1",
		authorizationEventMessage(t, "account-1", 7),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected Redis error, got %v", err)
	}
}

type fakeAdvancer struct {
	called    bool
	accountID string
	version   int64
	updated   bool
	err       error
}

func (a *fakeAdvancer) AdvanceAuthorizationVersion(
	_ context.Context,
	accountID string,
	version int64,
) (bool, error) {
	a.called = true
	a.accountID = accountID
	a.version = version
	return a.updated, a.err
}

func authorizationEventMessage(t *testing.T, accountID string, version int64) []byte {
	t.Helper()
	payload, err := json.Marshal(contractevents.IdentityAuthorizationChangedV1Payload{
		OperationID:          "operation-1",
		Action:               "identity.account.disabled",
		AuthorizationVersion: version,
	})
	if err != nil {
		t.Fatal(err)
	}
	message, err := json.Marshal(contractevents.Envelope{
		EventID:       "event-1",
		EventType:     contractevents.EventTypeIdentityAuthorizationChangedV1,
		Producer:      "identity-rpc",
		AggregateID:   accountID,
		SchemaVersion: 1,
		Payload:       payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	return message
}
