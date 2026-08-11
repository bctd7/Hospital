package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"hospital/service/identity/rpc/internal/authorizationprojection"
	"hospital/service/identity/rpc/internal/messaging/kafkaconsumer"
)

func TestRunAuthorizationVersionConsumerRetriesBeforeCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	record := &kgo.Record{
		Topic:     "identity.authorization.changed.v1",
		Partition: 1,
		Offset:    10,
		Key:       []byte("account-1"),
		Value:     []byte("event"),
	}
	consumer := &fakeAuthorizationConsumer{
		record:         record,
		commitFailures: 1,
		cancel:         cancel,
	}
	projector := &fakeAuthorizationProjector{failures: 1}

	runAuthorizationVersionConsumer(ctx, consumer, projector, time.Millisecond)

	if projector.calls != 2 {
		t.Fatalf("unexpected projector calls: got %d want 2", projector.calls)
	}
	if consumer.commitCalls != 2 {
		t.Fatalf("unexpected commit calls: got %d want 2", consumer.commitCalls)
	}
	if consumer.committed != record {
		t.Fatal("expected the successfully projected record to be committed")
	}
}

type fakeAuthorizationConsumer struct {
	record         *kgo.Record
	fetched        bool
	commitFailures int
	commitCalls    int
	committed      *kgo.Record
	cancel         context.CancelFunc
}

func (c *fakeAuthorizationConsumer) FetchMessage(ctx context.Context) (*kgo.Record, error) {
	if !c.fetched {
		c.fetched = true
		return c.record, nil
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (c *fakeAuthorizationConsumer) CommitMessage(_ context.Context, record *kgo.Record) error {
	c.commitCalls++
	if c.commitFailures > 0 {
		c.commitFailures--
		return errors.New("commit unavailable")
	}
	c.committed = record
	c.cancel()
	return nil
}

type fakeAuthorizationProjector struct {
	failures int
	calls    int
}

func (p *fakeAuthorizationProjector) Apply(
	_ context.Context,
	_ string,
	_ []byte,
) (bool, error) {
	p.calls++
	if p.failures > 0 {
		p.failures--
		return false, errors.New("redis unavailable")
	}
	return true, nil
}

var (
	_ authorizationMessageConsumer  = (*kafkaconsumer.Consumer)(nil)
	_ authorizationMessageProjector = (*authorizationprojection.Projector)(nil)
)
