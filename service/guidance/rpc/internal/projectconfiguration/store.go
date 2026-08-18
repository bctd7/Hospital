package projectconfiguration

import (
	"context"
	"time"

	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type Store interface {
	GetConfiguration(ctx context.Context, itemID string) (Configuration, error)
	ListRulesByOwner(ctx context.Context, itemID string) ([]precedence.Rule, error)
	WithinTransaction(ctx context.Context, fn func(TxStore) error) error
}

type TxStore interface {
	LockGraph(ctx context.Context) error
	GetTransactionForUpdate(ctx context.Context, transactionID string) (Transaction, bool, error)
	FindTransactionByOperation(ctx context.Context, operationID string) (Transaction, bool, error)
	CreateTransaction(ctx context.Context, transaction Transaction) error
	UpdateTransaction(ctx context.Context, transactionID, state, lastError string, retryCount int, updatedAt time.Time) error
	SetTransactionBefore(ctx context.Context, transactionID string, configuration *Configuration, rules []precedence.Rule, updatedAt time.Time) error
	GetConfigurationForUpdate(ctx context.Context, itemID string) (Configuration, bool, error)
	UpsertConfiguration(ctx context.Context, configuration Configuration) error
	DeleteConfiguration(ctx context.Context, itemID string) error
	ListAllRules(ctx context.Context) ([]precedence.Rule, error)
	ListRulesByOwnerForUpdate(ctx context.Context, itemID string) ([]precedence.Rule, error)
	DeleteRulesByOwner(ctx context.Context, itemID string) error
	InsertRule(ctx context.Context, rule precedence.Rule) error
}

type AppointmentParticipant interface {
	Prepare(ctx context.Context, transactionID string, command Command) error
	Confirm(ctx context.Context, transactionID string) (Project, error)
	Cancel(ctx context.Context, transactionID string) error
}

type ProjectDirectory interface {
	ResolveProject(ctx context.Context, itemID string) (Project, error)
}
