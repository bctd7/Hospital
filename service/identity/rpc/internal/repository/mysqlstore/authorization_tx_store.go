package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/account"
	"hospital/service/identity/rpc/internal/authorization"
)

var _ authorization.TxStore = (*mysqlAuthorizationTxStore)(nil)

type mysqlAuthorizationTxStore struct {
	tx *sql.Tx
}

func (s *mysqlAuthorizationTxStore) GetAuthorizationContext(ctx context.Context, accountID string) (authn.Principal, error) {
	return readAuthorizationContext(ctx, s.tx, accountID, true)
}

func (s *mysqlAuthorizationTxStore) FindOperation(ctx context.Context, operationID string) (authorization.Operation, bool, error) {
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

func (s *mysqlAuthorizationTxStore) SetRole(ctx context.Context, accountID, roleCode string) error {
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

func (s *mysqlAuthorizationTxStore) SetDepartment(ctx context.Context, accountID, departmentID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_staff_profiles
SET department_id = ?, updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, departmentID, accountID)
	if err != nil {
		return fmt.Errorf("change identity staff department: %w", err)
	}
	if err := requireAuthorizationAffected(result, "staff account"); err != nil {
		return err
	}
	return s.bumpAuthorizationVersion(ctx, accountID)
}

func (s *mysqlAuthorizationTxStore) SetAccountStatus(ctx context.Context, accountID, status string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET status = ?, authorization_version = authorization_version + 1, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, status, accountID)
	if err != nil {
		return fmt.Errorf("change identity account status: %w", err)
	}
	return requireAuthorizationAffected(result, "account")
}

func (s *mysqlAuthorizationTxStore) PromoteToDepartmentDoctor(ctx context.Context, accountID, departmentID string, verifyPhone bool) error {
	var departmentExists bool
	if err := s.tx.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM identity_organization_units
    WHERE id = ? AND unit_type = 'department' AND status = 'active'
)`, departmentID).Scan(&departmentExists); err != nil {
		return fmt.Errorf("check doctor department: %w", err)
	}
	if !departmentExists {
		return fmt.Errorf("%w: active department", authorization.ErrNotFound)
	}

	var phoneStatus string
	if err := s.tx.QueryRowContext(ctx, `
SELECT verification_status
FROM identity_account_phones
WHERE account_id = ?
FOR UPDATE`, accountID).Scan(&phoneStatus); errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: account phone", authorization.ErrNotFound)
	} else if err != nil {
		return fmt.Errorf("lock doctor phone: %w", err)
	}
	if phoneStatus != account.PhoneStatusVerified && !verifyPhone {
		return fmt.Errorf("%w: phone verification is required", authorization.ErrInvalid)
	}
	if verifyPhone {
		if _, err := s.tx.ExecContext(ctx, `
UPDATE identity_account_phones
SET verification_status = 'verified', verification_source = 'admin',
    verified_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, accountID); err != nil {
			return fmt.Errorf("verify doctor phone: %w", err)
		}
	}

	if _, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET account_type = 'staff', authorization_version = authorization_version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID); err != nil {
		return fmt.Errorf("promote staff account: %w", err)
	}
	if _, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_staff_profiles (account_id, department_id)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE department_id = VALUES(department_id), updated_at = CURRENT_TIMESTAMP(3)`, accountID, departmentID); err != nil {
		return fmt.Errorf("save doctor department: %w", err)
	}
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_account_roles (account_id, role_id)
SELECT ?, id FROM identity_roles WHERE code = 'department_doctor' AND status = 'active'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id), created_at = CURRENT_TIMESTAMP(3)`, accountID)
	if err != nil {
		return fmt.Errorf("assign department doctor role: %w", err)
	}
	var roleCode string
	if err := s.tx.QueryRowContext(ctx, `
SELECT r.code
FROM identity_account_roles ar
JOIN identity_roles r ON r.id = ar.role_id
WHERE ar.account_id = ?`, accountID).Scan(&roleCode); err != nil {
		return fmt.Errorf("verify department doctor role: %w", err)
	}
	if roleCode != authn.RoleDepartmentDoctor {
		return fmt.Errorf("%w: active department doctor role", authorization.ErrNotFound)
	}
	return nil
}

func (s *mysqlAuthorizationTxStore) RecordChange(ctx context.Context, change authorization.Change) error {
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

func (s *mysqlAuthorizationTxStore) bumpAuthorizationVersion(ctx context.Context, accountID string) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET authorization_version = authorization_version + 1, updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID)
	if err != nil {
		return fmt.Errorf("bump identity authorization version: %w", err)
	}
	return requireAuthorizationAffected(result, "account")
}

func requireAuthorizationAffected(result sql.Result, resource string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", authorization.ErrNotFound, resource)
	}
	return nil
}
