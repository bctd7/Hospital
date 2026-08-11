package kafkaconsumer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Consumer owns the Kafka client used to read authorization-version events.
// Offset commits are deliberately explicit: callers commit only after the
// corresponding Redis projection has been handled successfully.
type Consumer struct {
	client *kgo.Client
}

// New creates a Kafka consumer, joins the configured consumer group, and
// verifies that at least one broker is reachable before returning.
func New(
	ctx context.Context,
	brokers []string,
	clientID string,
	topic string,
	consumerGroup string,
) (*Consumer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("kafka client ID is required")
	}
	if strings.TrimSpace(topic) == "" {
		return nil, errors.New("kafka consumer topic is required")
	}
	if strings.TrimSpace(consumerGroup) == "" {
		return nil, errors.New("kafka consumer group is required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID(clientID),
		kgo.ConsumeTopics(topic),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka consumer: %w", err)
	}

	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping kafka broker: %w", err)
	}

	return &Consumer{client: client}, nil
}

// FetchMessage waits for the next available Kafka record. Fetching alone does
// not commit its offset because automatic commits are disabled in New.
func (c *Consumer) FetchMessage(ctx context.Context) (*kgo.Record, error) {
	for {
		fetches := c.client.PollRecords(ctx, 1)
		records := fetches.Records()
		if len(records) == 1 {
			return records[0], nil
		}
		if err := fetches.Err(); err != nil {
			return nil, fmt.Errorf("fetch kafka message: %w", err)
		}
	}
}

// CommitMessage records successful processing for this consumer group. If the
// process stops before this succeeds, Kafka can deliver the record again.
func (c *Consumer) CommitMessage(ctx context.Context, record *kgo.Record) error {
	if record == nil {
		return errors.New("kafka record is required")
	}
	if err := c.client.CommitRecords(ctx, record); err != nil {
		return fmt.Errorf("commit kafka message: %w", err)
	}
	return nil
}

// Close leaves the consumer group and closes the Kafka client connections.
func (c *Consumer) Close() {
	c.client.Close()
}
