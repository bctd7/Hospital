package version

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

func TestConsumerRetriesRedisAndCommitBeforeFetchingNextMessage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	record := &kgo.Record{
		Topic:     "identity.authorization.changed.v1",
		Partition: 1,
		Offset:    10,
		Key:       []byte("account-1"),
		Value:     authorizationEventMessage(t, "account-1", 7),
	}
	consumer := &fakeAuthorizationConsumer{
		record:         record,
		commitFailures: 1,
		cancel:         cancel,
	}
	versions := &retryingAdvancer{failures: 1}
	consumerWorker, err := NewConsumer(consumer, versions)
	if err != nil {
		t.Fatal(err)
	}

	consumerWorker.Run(ctx, time.Millisecond)

	if versions.calls != 2 {
		t.Fatalf("unexpected Redis version writes: got %d want 2", versions.calls)
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

type retryingAdvancer struct {
	failures int
	calls    int
}

func (p *retryingAdvancer) AdvanceAuthorizationVersion(
	_ context.Context,
	_ string,
	_ int64,
) (bool, error) {
	p.calls++
	if p.failures > 0 {
		p.failures--
		return false, errors.New("redis unavailable")
	}
	return true, nil
}
