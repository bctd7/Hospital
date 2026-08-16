package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hospital/service/guidance/rpc/internal/projectconfiguration"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

func (s *Store) GetConfiguration(ctx context.Context, itemID string) (projectconfiguration.Configuration, error) {
	configuration, found, err := getConfiguration(ctx, s.db, itemID, false)
	if err != nil {
		return projectconfiguration.Configuration{}, err
	}
	if !found {
		return projectconfiguration.Configuration{}, projectconfiguration.ErrNotFound
	}
	return configuration, nil
}

func (s *Store) ListRulesByOwner(ctx context.Context, itemID string) ([]precedence.Rule, error) {
	return listRulesByOwner(ctx, s.db, itemID, false)
}

func (s *Store) WithinTransaction(ctx context.Context, fn func(projectconfiguration.TxStore) error) error {
	if fn == nil {
		return errors.New("project configuration transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin project configuration transaction: %w", err)
	}
	if err = fn(&txStore{tx: tx}); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback project configuration transaction: %w", rollbackErr))
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit project configuration transaction: %w", err)
	}
	return nil
}

func (s *txStore) GetTransactionForUpdate(ctx context.Context, transactionID string) (projectconfiguration.Transaction, bool, error) {
	return getProjectConfigurationTransaction(s.tx.QueryRowContext(ctx, projectConfigurationTransactionSelect+" WHERE transaction_id = ? FOR UPDATE", transactionID))
}

func (s *txStore) FindTransactionByOperation(ctx context.Context, operationID string) (projectconfiguration.Transaction, bool, error) {
	return getProjectConfigurationTransaction(s.tx.QueryRowContext(ctx, projectConfigurationTransactionSelect+" WHERE operation_id = ?", operationID))
}

func (s *txStore) CreateTransaction(ctx context.Context, transaction projectconfiguration.Transaction) error {
	payload, err := json.Marshal(transaction.Payload)
	if err != nil {
		return fmt.Errorf("marshal project configuration payload: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO guidance_configuration_transactions (
    transaction_id, operation_id, action, item_id, operator_account_id, state,
    request_fingerprint, payload, last_error, retry_count, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		transaction.TransactionID, transaction.OperationID, transaction.Action, transaction.ItemID,
		transaction.OperatorAccountID, transaction.State, transaction.RequestFingerprint, payload,
		transaction.LastError, transaction.RetryCount, transaction.CreatedAt, transaction.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create guidance configuration transaction: %w", err)
	}
	return nil
}

func (s *txStore) UpdateTransaction(ctx context.Context, transactionID, state, lastError string, retryCount int, updatedAt time.Time) error {
	_, err := s.tx.ExecContext(ctx, `
UPDATE guidance_configuration_transactions
SET state = ?, last_error = ?, retry_count = ?, updated_at = ?
WHERE transaction_id = ?`, state, lastError, retryCount, updatedAt, transactionID)
	if err != nil {
		return fmt.Errorf("update guidance configuration transaction: %w", err)
	}
	return nil
}

func (s *txStore) SetTransactionBefore(ctx context.Context, transactionID string, configuration *projectconfiguration.Configuration, rules []precedence.Rule, updatedAt time.Time) error {
	var configurationJSON any
	if configuration != nil {
		encoded, err := json.Marshal(configuration)
		if err != nil {
			return fmt.Errorf("marshal previous guidance configuration: %w", err)
		}
		configurationJSON = encoded
	}
	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return fmt.Errorf("marshal previous precedence rules: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
UPDATE guidance_configuration_transactions
SET before_configuration = ?, before_precedence_rules = ?, updated_at = ?
WHERE transaction_id = ?`, configurationJSON, rulesJSON, updatedAt, transactionID)
	if err != nil {
		return fmt.Errorf("save guidance transaction snapshot: %w", err)
	}
	return nil
}

func (s *txStore) GetConfigurationForUpdate(ctx context.Context, itemID string) (projectconfiguration.Configuration, bool, error) {
	return getConfiguration(ctx, s.tx, itemID, true)
}

func (s *txStore) UpsertConfiguration(ctx context.Context, configuration projectconfiguration.Configuration) error {
	rules, err := json.Marshal(configuration.PreparationRules)
	if err != nil {
		return fmt.Errorf("marshal preparation rules: %w", err)
	}
	reminders, err := json.Marshal(configuration.Reminders)
	if err != nil {
		return fmt.Errorf("marshal patient reminders: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO guidance_item_configurations (
    item_id, description, preparation_rules, reminders, version, updated_by, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    description = VALUES(description), preparation_rules = VALUES(preparation_rules), reminders = VALUES(reminders),
    version = VALUES(version), updated_by = VALUES(updated_by), updated_at = VALUES(updated_at)`,
		configuration.ItemID, configuration.Description, rules, reminders, configuration.Version,
		configuration.UpdatedBy, configuration.CreatedAt, configuration.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert guidance item configuration: %w", err)
	}
	return nil
}

func (s *txStore) DeleteConfiguration(ctx context.Context, itemID string) error {
	if _, err := s.tx.ExecContext(ctx, "DELETE FROM guidance_item_configurations WHERE item_id = ?", itemID); err != nil {
		return fmt.Errorf("delete guidance item configuration: %w", err)
	}
	return nil
}

func (s *txStore) ListAllRules(ctx context.Context) ([]precedence.Rule, error) {
	return listRules(ctx, s.tx)
}

func (s *txStore) ListRulesByOwnerForUpdate(ctx context.Context, itemID string) ([]precedence.Rule, error) {
	return listRulesByOwner(ctx, s.tx, itemID, true)
}

func (s *txStore) DeleteRulesByOwner(ctx context.Context, itemID string) error {
	if _, err := s.tx.ExecContext(ctx, "DELETE FROM guidance_precedence_rules WHERE owner_item_id = ?", itemID); err != nil {
		return fmt.Errorf("delete owned precedence rules: %w", err)
	}
	return nil
}

func (s *txStore) InsertRule(ctx context.Context, rule precedence.Rule) error {
	_, err := s.Insert(ctx, rule)
	return err
}

type configurationQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getConfiguration(ctx context.Context, db configurationQueryer, itemID string, forUpdate bool) (projectconfiguration.Configuration, bool, error) {
	query := `
SELECT item_id, description, preparation_rules, reminders, version, updated_by, created_at, updated_at
FROM guidance_item_configurations WHERE item_id = ?`
	if forUpdate {
		query += " FOR UPDATE"
	}
	var configuration projectconfiguration.Configuration
	var rulesJSON, remindersJSON []byte
	err := db.QueryRowContext(ctx, query, itemID).Scan(
		&configuration.ItemID, &configuration.Description, &rulesJSON, &remindersJSON, &configuration.Version,
		&configuration.UpdatedBy, &configuration.CreatedAt, &configuration.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return projectconfiguration.Configuration{}, false, nil
	}
	if err != nil {
		return projectconfiguration.Configuration{}, false, fmt.Errorf("get guidance item configuration: %w", err)
	}
	if err := json.Unmarshal(rulesJSON, &configuration.PreparationRules); err != nil {
		return projectconfiguration.Configuration{}, false, fmt.Errorf("decode preparation rules: %w", err)
	}
	if err := json.Unmarshal(remindersJSON, &configuration.Reminders); err != nil {
		return projectconfiguration.Configuration{}, false, fmt.Errorf("decode patient reminders: %w", err)
	}
	return configuration, true, nil
}

func listRulesByOwner(ctx context.Context, db queryer, itemID string, forUpdate bool) ([]precedence.Rule, error) {
	query := ruleSelect + " WHERE owner_item_id = ? ORDER BY predecessor_item_id, successor_item_id"
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := db.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, fmt.Errorf("list owned precedence rules: %w", err)
	}
	defer rows.Close()
	rules := make([]precedence.Rule, 0)
	for rows.Next() {
		rule, scanErr := scanRule(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan owned precedence rule: %w", scanErr)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owned precedence rules: %w", err)
	}
	return rules, nil
}

const projectConfigurationTransactionSelect = `
SELECT transaction_id, operation_id, action, item_id, operator_account_id, state,
       request_fingerprint, payload, before_configuration, before_precedence_rules,
       last_error, retry_count, created_at, updated_at
FROM guidance_configuration_transactions`

func getProjectConfigurationTransaction(row *sql.Row) (projectconfiguration.Transaction, bool, error) {
	var transaction projectconfiguration.Transaction
	var payload []byte
	var beforeConfiguration, beforeRules []byte
	err := row.Scan(
		&transaction.TransactionID, &transaction.OperationID, &transaction.Action, &transaction.ItemID,
		&transaction.OperatorAccountID, &transaction.State, &transaction.RequestFingerprint, &payload,
		&beforeConfiguration, &beforeRules, &transaction.LastError, &transaction.RetryCount,
		&transaction.CreatedAt, &transaction.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return projectconfiguration.Transaction{}, false, nil
	}
	if err != nil {
		return projectconfiguration.Transaction{}, false, fmt.Errorf("get guidance configuration transaction: %w", err)
	}
	if err := json.Unmarshal(payload, &transaction.Payload); err != nil {
		return projectconfiguration.Transaction{}, false, fmt.Errorf("decode guidance transaction payload: %w", err)
	}
	if len(beforeConfiguration) > 0 {
		var configuration projectconfiguration.Configuration
		if err := json.Unmarshal(beforeConfiguration, &configuration); err != nil {
			return projectconfiguration.Transaction{}, false, fmt.Errorf("decode previous guidance configuration: %w", err)
		}
		transaction.BeforeConfiguration = &configuration
	}
	if len(beforeRules) > 0 {
		if err := json.Unmarshal(beforeRules, &transaction.BeforePrecedenceRules); err != nil {
			return projectconfiguration.Transaction{}, false, fmt.Errorf("decode previous precedence rules: %w", err)
		}
	}
	return transaction, true, nil
}

var _ projectconfiguration.Store = (*Store)(nil)
var _ projectconfiguration.TxStore = (*txStore)(nil)
