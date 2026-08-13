package version

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	contractevents "hospital/contracts/events"
)

func TestConsumerAppliesAuthorizationVersionEvent(t *testing.T) {
	advancer := &fakeAdvancer{updated: true}
	consumer, err := NewConsumer(fakeMessageReader{}, advancer)
	if err != nil {
		t.Fatal(err)
	}

	message := authorizationEventMessage(t, "account-1", 7)
	updated, err := consumer.apply(context.Background(), "account-1", message)
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("expected authorization version to be updated")
	}
	if advancer.accountID != "account-1" || advancer.version != 7 {
		t.Fatalf("unexpected version update: account=%q version=%d", advancer.accountID, advancer.version)
	}
}

func TestConsumerRejectsMismatchedKafkaKey(t *testing.T) {
	advancer := &fakeAdvancer{}
	consumer, err := NewConsumer(fakeMessageReader{}, advancer)
	if err != nil {
		t.Fatal(err)
	}

	_, err = consumer.apply(
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

func TestConsumerReturnsRedisFailure(t *testing.T) {
	wantErr := errors.New("redis unavailable")
	consumer, err := NewConsumer(fakeMessageReader{}, &fakeAdvancer{err: wantErr})
	if err != nil {
		t.Fatal(err)
	}

	_, err = consumer.apply(
		context.Background(),
		"account-1",
		authorizationEventMessage(t, "account-1", 7),
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected Redis error, got %v", err)
	}
}

func TestConsumerRetriesRedisAndCommitBeforeFetchingNextMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	record := &kgo.Record{
		Topic: "identity.authorization.changed.v1", Partition: 1, Offset: 10,
		Key: []byte("account-1"), Value: authorizationEventMessage(t, "account-1", 7),
	}
	messageReader := &retryingMessageReader{record: record, commitFailures: 1, cancel: cancel}
	versions := &retryingVersionStore{failures: 1}
	consumer, err := NewConsumer(messageReader, versions)
	if err != nil {
		t.Fatal(err)
	}
	consumer.Run(ctx, time.Millisecond)
	if versions.calls != 2 {
		t.Fatalf("unexpected Redis version writes: got %d want 2", versions.calls)
	}
	if messageReader.commitCalls != 2 || messageReader.committed != record {
		t.Fatalf("unexpected commit result: calls=%d record=%p", messageReader.commitCalls, messageReader.committed)
	}
}

func TestBuildChangedEvent(t *testing.T) {
	accountID := "20000000-0000-0000-0000-000000000002"
	operationID := "20000000-0000-0000-0000-000000000020"
	event, changed, err := BuildChangedEvent(accountID, operationID, "identity.doctor.promoted", 7, 8)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || event.AggregateID != accountID || event.EventType != contractevents.EventTypeIdentityAuthorizationChangedV1 || event.SchemaVersion != 1 {
		t.Fatalf("unexpected authorization event: %#v", event)
	}
	var payload contractevents.IdentityAuthorizationChangedV1Payload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OperationID != operationID || payload.AuthorizationVersion != 8 {
		t.Fatalf("unexpected authorization payload: %#v", payload)
	}
}

func TestBuildChangedEventSkipsProfileOnlyChange(t *testing.T) {
	event, changed, err := BuildChangedEvent("account-1", "operation-1", "identity.doctor.updated", 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if changed || event.AggregateID != "" || len(event.Payload) != 0 {
		t.Fatalf("profile-only change created an authorization event: %#v", event)
	}
}

func TestBuildChangedEventRejectsVersionDecrease(t *testing.T) {
	_, changed, err := BuildChangedEvent("account-1", "operation-1", "identity.account.changed", 8, 7)
	if err == nil || changed {
		t.Fatalf("expected version decrease to be rejected: changed=%t err=%v", changed, err)
	}
}

type fakeMessageReader struct{}

func (fakeMessageReader) FetchMessage(context.Context) (*kgo.Record, error) {
	return nil, errors.New("unused")
}
func (fakeMessageReader) CommitMessage(context.Context, *kgo.Record) error { return nil }

type fakeAdvancer struct {
	called    bool
	accountID string
	version   int64
	updated   bool
	err       error
}

type retryingMessageReader struct {
	record         *kgo.Record
	fetched        bool
	commitFailures int
	commitCalls    int
	committed      *kgo.Record
	cancel         context.CancelFunc
}

func (r *retryingMessageReader) FetchMessage(ctx context.Context) (*kgo.Record, error) {
	if !r.fetched {
		r.fetched = true
		return r.record, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (r *retryingMessageReader) CommitMessage(_ context.Context, record *kgo.Record) error {
	r.commitCalls++
	if r.commitFailures > 0 {
		r.commitFailures--
		return errors.New("commit unavailable")
	}
	r.committed = record
	r.cancel()
	return nil
}

type retryingVersionStore struct {
	failures int
	calls    int
}

func (s *retryingVersionStore) AdvanceAuthorizationVersion(context.Context, string, int64) (bool, error) {
	s.calls++
	if s.failures > 0 {
		s.failures--
		return false, errors.New("redis unavailable")
	}
	return true, nil
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
