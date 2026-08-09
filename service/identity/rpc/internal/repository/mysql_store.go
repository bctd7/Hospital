package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/authorization"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(dataSource string) (*MySQLStore, error) {
	if dataSource == "" {
		return nil, errors.New("identity mysql data source is required")
	}
	db, err := sql.Open("mysql", dataSource)
	if err != nil {
		return nil, fmt.Errorf("open identity mysql: %w", err)
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping identity mysql: %w", err)
	}
	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error) {
	return readAuthorizationContext(ctx, s.db, accountID, false)
}

func (s *MySQLStore) WithinTransaction(ctx context.Context, fn func(authorization.TxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin identity transaction: %w", err)
	}
	store := &mysqlTxStore{tx: tx}
	if err := fn(store); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback identity transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit identity transaction: %w", err)
	}
	return nil
}

type mysqlTxStore struct {
	tx *sql.Tx
}

func (s *mysqlTxStore) GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error) {
	return readAuthorizationContext(ctx, s.tx, accountID, true)
}

func (s *mysqlTxStore) FindOperation(ctx context.Context, operationID string) (authorization.Operation, bool, error) {
	var operation authorization.Operation
	err := s.tx.QueryRowContext(ctx, `
SELECT operator_account_id, target_account_id
FROM identity_authorization_audit
WHERE operation_id = ?`, operationID).Scan(&operation.OperatorAccountID, &operation.TargetAccountID)
	if errors.Is(err, sql.ErrNoRows) {
		return authorization.Operation{}, false, nil
	}
	if err != nil {
		return authorization.Operation{}, false, fmt.Errorf("find authorization operation: %w", err)
	}
	return operation, true, nil
}

func (s *mysqlTxStore) SetRole(ctx context.Context, accountID, roleCode string) error {
	result, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_account_roles (account_id, role_id)
SELECT ?, id FROM identity_roles WHERE code = ? AND status = 'active'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id), created_at = CURRENT_TIMESTAMP(3)`, accountID, roleCode)
	if err != nil {
		return fmt.Errorf("assign identity role: %w", err)
	}
	_ = result // assigning the current role is a valid audited no-op
	return s.bumpAuthorizationVersion(ctx, accountID)
}

func (s *mysqlTxStore) SetDepartment(ctx context.Context, accountID, departmentID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_staff_profiles
SET department_id = ?, updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, departmentID, accountID)
	if err != nil {
		return fmt.Errorf("change identity staff department: %w", err)
	}
	if err := requireAffected(result, "staff account"); err != nil {
		return err
	}
	return s.bumpAuthorizationVersion(ctx, accountID)
}

func (s *mysqlTxStore) SetAccountStatus(ctx context.Context, accountID, status string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET status = ?, authorization_version = authorization_version + 1, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, status, accountID)
	if err != nil {
		return fmt.Errorf("change identity account status: %w", err)
	}
	return requireAffected(result, "account")
}

func (s *mysqlTxStore) RecordChange(ctx context.Context, change authorization.Change) error {
	beforeData, err := json.Marshal(change.Before)
	if err != nil {
		return fmt.Errorf("marshal authorization before state: %w", err)
	}
	afterData, err := json.Marshal(change.After)
	if err != nil {
		return fmt.Errorf("marshal authorization after state: %w", err)
	}

	auditID := uuid.NewString()
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO identity_authorization_audit
    (id, operation_id, operator_account_id, target_account_id, action, before_data, after_data, request_id)
VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`,
		auditID, change.OperationID, change.OperatorAccountID, change.TargetAccountID,
		change.Action, beforeData, afterData, change.RequestID)
	if err != nil {
		return fmt.Errorf("insert identity authorization audit: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"operation_id":          change.OperationID,
		"operator_account_id":   change.OperatorAccountID,
		"target_account_id":     change.TargetAccountID,
		"action":                change.Action,
		"authorization_version": change.After.AuthorizationVersion,
	})
	if err != nil {
		return fmt.Errorf("marshal identity authorization event: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO identity_outbox_events
    (event_id, aggregate_id, event_type, schema_version, payload, occurred_at)
VALUES (?, ?, 'identity.authorization.changed.v1', 1, ?, ?)`,
		uuid.NewString(), change.TargetAccountID, payload, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert identity authorization outbox: %w", err)
	}
	return nil
}

func (s *mysqlTxStore) bumpAuthorizationVersion(ctx context.Context, accountID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET authorization_version = authorization_version + 1, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID)
	if err != nil {
		return fmt.Errorf("bump identity authorization version: %w", err)
	}
	return requireAffected(result, "account")
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func readAuthorizationContext(ctx context.Context, db queryer, accountID string, forUpdate bool) (authn.Principal, error) {
	query := `
SELECT
    a.id,
    a.account_type,
    a.status,
    a.authorization_version,
    COALESCE(sp.department_id, ''),
    COALESCE(r.code, ''),
    COALESCE(p.code, '')
FROM identity_accounts a
LEFT JOIN identity_staff_profiles sp ON sp.account_id = a.id
LEFT JOIN identity_account_roles ar ON ar.account_id = a.id
LEFT JOIN identity_roles r ON r.id = ar.role_id AND r.status = 'active'
LEFT JOIN identity_role_permissions rp ON rp.role_id = r.id
LEFT JOIN identity_permissions p ON p.id = rp.permission_id AND p.status = 'active'
WHERE a.id = ?`
	if forUpdate {
		query += " FOR UPDATE"
	}
	rows, err := db.QueryContext(ctx, query, accountID)
	if err != nil {
		return authn.Principal{}, fmt.Errorf("query identity authorization context: %w", err)
	}
	defer rows.Close()

	var principal authn.Principal
	roles := make(map[string]struct{})
	permissions := make(map[string]struct{})
	found := false
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(
			&principal.AccountID, &principal.AccountType, &principal.Status,
			&principal.AuthorizationVersion, &principal.DepartmentID, &role, &permission,
		); err != nil {
			return authn.Principal{}, fmt.Errorf("scan identity authorization context: %w", err)
		}
		found = true
		if role != "" {
			roles[role] = struct{}{}
		}
		if permission != "" {
			permissions[permission] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return authn.Principal{}, fmt.Errorf("iterate identity authorization context: %w", err)
	}
	if !found {
		return authn.Principal{}, authorization.ErrNotFound
	}
	principal.Roles = sortedKeys(roles)
	principal.Permissions = sortedKeys(permissions)
	return principal, nil
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func requireAffected(result sql.Result, resource string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", authorization.ErrNotFound, resource)
	}
	return nil
}
