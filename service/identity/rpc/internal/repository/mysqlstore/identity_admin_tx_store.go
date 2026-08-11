package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/service/identity/rpc/internal/identityadmin"
)

var _ identityadmin.TxStore = (*mysqlIdentityAdminTxStore)(nil)

type mysqlIdentityAdminTxStore struct {
	tx *sql.Tx
}

func (s *Store) WithinIdentityAdminTransaction(ctx context.Context, fn func(identityadmin.TxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin identity admin transaction: %w", err)
	}
	defer tx.Rollback()
	if err := fn(&mysqlIdentityAdminTxStore{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit identity admin transaction: %w", err)
	}
	return nil
}

func (s *mysqlIdentityAdminTxStore) FindIdentityAdminOperation(ctx context.Context, operationID string) (identityadmin.Operation, bool, error) {
	var operation identityadmin.Operation
	err := s.tx.QueryRowContext(ctx, `
SELECT operator_account_id, target_account_id, action
FROM identity_authorization_audit
WHERE operation_id = ?`, operationID).Scan(
		&operation.OperatorAccountID, &operation.TargetAccountID, &operation.Action,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return identityadmin.Operation{}, false, nil
	}
	if err != nil {
		return identityadmin.Operation{}, false, fmt.Errorf("find identity admin operation: %w", err)
	}
	return operation, true, nil
}

func (s *mysqlIdentityAdminTxStore) GetIdentityAdminAccountForUpdate(ctx context.Context, accountID string) (identityadmin.Account, error) {
	return scanIdentityAdminAccount(s.tx.QueryRowContext(ctx, identityAdminAccountSelect+" WHERE a.id = ? FOR UPDATE", accountID))
}

func (s *mysqlIdentityAdminTxStore) PromoteDoctor(
	ctx context.Context,
	accountID, departmentID string,
	profile identityadmin.DoctorProfileInput,
	expectedVersion int64,
	verifyPhone bool,
) error {
	if err := s.requireActiveDepartment(ctx, departmentID); err != nil {
		return err
	}
	var phoneStatus string
	err := s.tx.QueryRowContext(ctx, `
SELECT verification_status
FROM identity_account_phones
WHERE account_id = ?
FOR UPDATE`, accountID).Scan(&phoneStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: account phone", identityadmin.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("lock promoted doctor phone: %w", err)
	}
	if phoneStatus != "verified" && !verifyPhone {
		return fmt.Errorf("%w: verified phone is required", identityadmin.ErrInvalidState)
	}
	if verifyPhone {
		if _, err := s.tx.ExecContext(ctx, `
UPDATE identity_account_phones
SET verification_status = 'verified', verification_source = 'admin',
    verified_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, accountID); err != nil {
			return fmt.Errorf("verify promoted doctor phone: %w", err)
		}
	}
	if err := s.bumpManagedAccount(ctx, accountID, expectedVersion, true, "'staff'"); err != nil {
		return err
	}
	if _, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_staff_profiles
    (account_id, department_id, staff_no, display_name, avatar_url, description, staff_status)
VALUES (?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), 'active')
ON DUPLICATE KEY UPDATE
    department_id = VALUES(department_id),
    staff_no = VALUES(staff_no),
    display_name = VALUES(display_name),
    avatar_url = VALUES(avatar_url),
    description = VALUES(description),
    staff_status = 'active',
    updated_at = CURRENT_TIMESTAMP(3)`,
		accountID, departmentID, profile.StaffNo, profile.DisplayName, profile.AvatarURL, profile.Description); err != nil {
		return fmt.Errorf("save promoted doctor profile: %w", err)
	}
	if _, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_account_roles (account_id, role_id)
SELECT ?, id FROM identity_roles WHERE code = 'department_doctor' AND status = 'active'
ON DUPLICATE KEY UPDATE role_id = VALUES(role_id), created_at = CURRENT_TIMESTAMP(3)`, accountID); err != nil {
		return fmt.Errorf("assign promoted doctor role: %w", err)
	}
	return nil
}

func (s *mysqlIdentityAdminTxStore) UpdateDoctor(ctx context.Context, accountID string, profile identityadmin.OptionalDoctorProfileInput, expectedVersion int64) error {
	if err := s.bumpManagedAccount(ctx, accountID, expectedVersion, false, "account_type"); err != nil {
		return err
	}
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_staff_profiles
SET display_name = IF(?, ?, display_name),
    staff_no = IF(?, NULLIF(?, ''), staff_no),
    avatar_url = IF(?, NULLIF(?, ''), avatar_url),
    description = IF(?, NULLIF(?, ''), description),
    updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ? AND staff_status = 'active'`,
		profile.DisplayName != nil, optionalString(profile.DisplayName),
		profile.StaffNo != nil, optionalString(profile.StaffNo),
		profile.AvatarURL != nil, optionalString(profile.AvatarURL),
		profile.Description != nil, optionalString(profile.Description), accountID)
	if err != nil {
		return fmt.Errorf("update doctor profile: %w", err)
	}
	return requireIdentityAdminAffected(result, "active doctor")
}

func (s *mysqlIdentityAdminTxStore) ChangeDoctorDepartment(ctx context.Context, accountID, departmentID string, expectedVersion int64) error {
	if err := s.requireActiveDepartment(ctx, departmentID); err != nil {
		return err
	}
	if err := s.bumpManagedAccount(ctx, accountID, expectedVersion, true, "account_type"); err != nil {
		return err
	}
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_staff_profiles
SET department_id = ?, updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ? AND staff_status = 'active'`, departmentID, accountID)
	if err != nil {
		return fmt.Errorf("change managed doctor department: %w", err)
	}
	return requireIdentityAdminAffected(result, "active doctor")
}

func (s *mysqlIdentityAdminTxStore) RevokeDoctor(ctx context.Context, accountID string, expectedVersion int64) error {
	if err := s.bumpManagedAccount(ctx, accountID, expectedVersion, true, "'patient'"); err != nil {
		return err
	}
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_staff_profiles
SET staff_status = 'revoked', updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ? AND staff_status = 'active'`, accountID)
	if err != nil {
		return fmt.Errorf("revoke doctor profile: %w", err)
	}
	if err := requireIdentityAdminAffected(result, "active doctor"); err != nil {
		return err
	}
	if _, err := s.tx.ExecContext(ctx, `DELETE FROM identity_account_roles WHERE account_id = ?`, accountID); err != nil {
		return fmt.Errorf("remove revoked doctor role: %w", err)
	}
	return nil
}

func (s *mysqlIdentityAdminTxStore) SetManagedAccountStatus(ctx context.Context, accountID, status string, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_accounts
SET status = ?,
    management_version = management_version + 1,
    authorization_version = authorization_version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND management_version = ?`, status, accountID, expectedVersion)
	if err != nil {
		return fmt.Errorf("set managed account status: %w", err)
	}
	return requireIdentityAdminVersionedMutation(result)
}

func (s *mysqlIdentityAdminTxStore) RecordIdentityAdminChange(ctx context.Context, change identityadmin.Change) error {
	beforeData, err := json.Marshal(change.Before)
	if err != nil {
		return fmt.Errorf("marshal identity admin before state: %w", err)
	}
	afterData, err := json.Marshal(change.After)
	if err != nil {
		return fmt.Errorf("marshal identity admin after state: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO identity_authorization_audit
    (id, operation_id, operator_account_id, target_account_id, action, before_data, after_data, request_id)
VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`,
		uuid.NewString(), change.OperationID, change.OperatorAccountID, change.TargetAccountID,
		change.Action, beforeData, afterData, change.RequestID)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: operation_id already exists", identityadmin.ErrConflict)
		}
		return fmt.Errorf("insert identity admin audit: %w", err)
	}
	payload, err := json.Marshal(map[string]any{
		"operation_id":          change.OperationID,
		"operator_account_id":   change.OperatorAccountID,
		"target_account_id":     change.TargetAccountID,
		"action":                change.Action,
		"management_version":    change.After.ManagementVersion,
		"authorization_version": change.After.AuthorizationVersion,
		"before_department_id":  change.Before.DepartmentID,
		"after_department_id":   change.After.DepartmentID,
	})
	if err != nil {
		return fmt.Errorf("marshal identity admin event: %w", err)
	}
	if _, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_outbox_events
    (event_id, aggregate_id, event_type, schema_version, payload, occurred_at)
VALUES (?, ?, ?, 1, ?, ?)`, uuid.NewString(), change.TargetAccountID,
		change.Action, payload, time.Now().UTC()); err != nil {
		return fmt.Errorf("insert identity admin outbox: %w", err)
	}
	return nil
}

func (s *mysqlIdentityAdminTxStore) requireActiveDepartment(ctx context.Context, departmentID string) error {
	var exists bool
	if err := s.tx.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1 FROM identity_organization_units
    WHERE id = ? AND unit_type = 'department' AND status = 'active'
)`, departmentID).Scan(&exists); err != nil {
		return fmt.Errorf("check managed doctor department: %w", err)
	}
	if !exists {
		return fmt.Errorf("%w: active department", identityadmin.ErrNotFound)
	}
	return nil
}

func (s *mysqlIdentityAdminTxStore) bumpManagedAccount(ctx context.Context, accountID string, expectedVersion int64, authorizationChange bool, accountTypeExpression string) error {
	authorizationIncrement := 0
	if authorizationChange {
		authorizationIncrement = 1
	}
	query := `
UPDATE identity_accounts
SET account_type = ` + accountTypeExpression + `,
    management_version = management_version + 1,
    authorization_version = authorization_version + ?,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND management_version = ?`
	result, err := s.tx.ExecContext(ctx, query, authorizationIncrement, accountID, expectedVersion)
	if err != nil {
		return fmt.Errorf("bump managed account version: %w", err)
	}
	return requireIdentityAdminVersionedMutation(result)
}

func requireIdentityAdminVersionedMutation(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read identity admin affected rows: %w", err)
	}
	if affected == 0 {
		return identityadmin.ErrVersionConflict
	}
	return nil
}

func requireIdentityAdminAffected(result sql.Result, resource string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read identity admin affected rows: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", identityadmin.ErrNotFound, resource)
	}
	return nil
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
