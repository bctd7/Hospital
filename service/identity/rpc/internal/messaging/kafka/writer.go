package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Writer owns the Kafka client used to publish integration events.
type Writer struct {
	client *kgo.Client
}

// New creates a Kafka producer and verifies that at least one configured
// broker is reachable before returning it to the service.
func NewWriter(ctx context.Context, brokers []string, clientID string) (*Writer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("kafka brokers are required")
	}
	if clientID == "" {
		return nil, errors.New("kafka client ID is required")
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID(clientID),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}

	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping kafka broker: %w", err)
	}

	return &Writer{client: client}, nil
}

func (p *Writer) Publish(
	ctx context.Context,
	topic string,
	key string,
	message []byte,
) error {
	if topic == "" {
		return errors.New("kafka topic is required")
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: message,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("publish kafka message: %w", err)
	}

	return nil
}

// Close flushes and closes the Kafka client connections.
func (p *Writer) Close() {
	p.client.Close()
}
