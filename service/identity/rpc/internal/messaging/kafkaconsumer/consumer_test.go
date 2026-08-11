package kafkaconsumer

import (
	"context"
	"strings"
	"testing"
)

func TestNewValidatesRequiredConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		brokers       []string
		clientID      string
		topic         string
		consumerGroup string
		wantError     string
	}{
		{name: "brokers", clientID: "identity-rpc", topic: "topic", consumerGroup: "group", wantError: "brokers"},
		{name: "client id", brokers: []string{"localhost:9092"}, topic: "topic", consumerGroup: "group", wantError: "client ID"},
		{name: "topic", brokers: []string{"localhost:9092"}, clientID: "identity-rpc", consumerGroup: "group", wantError: "topic"},
		{name: "consumer group", brokers: []string{"localhost:9092"}, clientID: "identity-rpc", topic: "topic", wantError: "consumer group"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer, err := New(
				context.Background(),
				tt.brokers,
				tt.clientID,
				tt.topic,
				tt.consumerGroup,
			)
			if consumer != nil {
				consumer.Close()
				t.Fatal("expected consumer creation to fail")
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
