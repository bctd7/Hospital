package svc

import (
	"context"
	"fmt"
	"strings"
	"time"

	commonauthversion "hospital/common/authz/version"
	contractevents "hospital/contracts/events"
	identityauthversion "hospital/service/identity/rpc/internal/authorization/version"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/messaging/kafka"
	"hospital/service/identity/rpc/internal/messaging/outbox"
	"hospital/service/identity/rpc/internal/repository/mysqlstore"
)

type messagingRuntime struct {
	workers Workers
	writer  *kafka.Writer
	reader  *kafka.Reader
}

func buildMessaging(c config.Config, store *mysqlstore.Store, versions commonauthversion.AuthorizationVersionAdvancer) (messagingRuntime, error) {
	if !c.Kafka.Enabled {
		return messagingRuntime{}, nil
	}
	brokers := splitCommaSeparated(c.Kafka.Brokers)
	writerCtx, writerCancel := context.WithTimeout(context.Background(), 5*time.Second)
	writer, err := kafka.NewWriter(writerCtx, brokers, c.Kafka.ClientID)
	writerCancel()
	if err != nil {
		return messagingRuntime{}, fmt.Errorf("create identity kafka writer: %w", err)
	}
	publisher, err := outbox.NewPublisher(
		store, writer, c.Kafka.AuthorizationChangedTopic,
		contractevents.EventTypeIdentityAuthorizationChangedV1,
		c.Kafka.ClientID, c.Kafka.BatchSize,
	)
	if err != nil {
		writer.Close()
		return messagingRuntime{}, fmt.Errorf("create identity outbox publisher: %w", err)
	}
	readerCtx, readerCancel := context.WithTimeout(context.Background(), 5*time.Second)
	reader, err := kafka.NewReader(readerCtx, brokers, c.Kafka.ClientID, c.Kafka.AuthorizationChangedTopic, c.Kafka.AuthorizationVersionConsumerGroup)
	readerCancel()
	if err != nil {
		writer.Close()
		return messagingRuntime{}, fmt.Errorf("create identity kafka reader: %w", err)
	}
	consumer, err := identityauthversion.NewConsumer(reader, versions)
	if err != nil {
		reader.Close()
		writer.Close()
		return messagingRuntime{}, fmt.Errorf("create authorization version consumer: %w", err)
	}
	return messagingRuntime{
		workers: Workers{OutboxPublisher: publisher, AuthorizationVersionConsumer: consumer},
		writer:  writer, reader: reader,
	}, nil
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
