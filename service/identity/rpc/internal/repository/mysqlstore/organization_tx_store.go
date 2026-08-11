package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"hospital/service/identity/rpc/internal/organization"
)

var _ organization.TxStore = (*mysqlOrganizationTxStore)(nil)

type mysqlOrganizationTxStore struct {
	tx *sql.Tx
}

func (s *mysqlOrganizationTxStore) FindOperation(ctx context.Context, operationID string) (organization.Operation, bool, error) {
	var operation organization.Operation
	var resultData []byte
	err := s.tx.QueryRowContext(ctx, `
SELECT operator_account_id, unit_id, action, after_data
FROM identity_organization_audit
WHERE operation_id = ?`, operationID).Scan(
		&operation.OperatorAccountID,
		&operation.UnitID,
		&operation.Action,
		&resultData,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return organization.Operation{}, false, nil
	}
	if err != nil {
		return organization.Operation{}, false, fmt.Errorf("find organization operation: %w", err)
	}
	if err := json.Unmarshal(resultData, &operation.Result); err != nil {
		return organization.Operation{}, false, fmt.Errorf("decode organization operation result: %w", err)
	}
	return operation, true, nil
}

func (s *mysqlOrganizationTxStore) GetUnitForUpdate(ctx context.Context, unitID string) (organization.Unit, error) {
	var unit organization.Unit
	err := s.tx.QueryRowContext(ctx, `
SELECT id, COALESCE(parent_id, ''), unit_type, code, name, status, version
FROM identity_organization_units
WHERE id = ?
FOR UPDATE`, unitID).Scan(
		&unit.ID,
		&unit.ParentID,
		&unit.Type,
		&unit.Code,
		&unit.Name,
		&unit.Status,
		&unit.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return organization.Unit{}, organization.ErrNotFound
	}
	if err != nil {
		return organization.Unit{}, fmt.Errorf("lock organization unit: %w", err)
	}

	switch unit.Type {
	case organization.UnitTypeCampus:
		unit.ChildCount, err = s.CountActiveChildren(ctx, unit.ID)
	case organization.UnitTypeDepartment:
		unit.DoctorCount, err = s.CountActiveDoctors(ctx, unit.ID)
	}
	if err != nil {
		return organization.Unit{}, err
	}
	return unit, nil
}

func (s *mysqlOrganizationTxStore) CreateUnit(ctx context.Context, unit organization.Unit) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO identity_organization_units
    (id, parent_id, unit_type, code, name, status, version)
VALUES (?, NULLIF(?, ''), ?, ?, ?, ?, ?)`,
		unit.ID, unit.ParentID, unit.Type, unit.Code, unit.Name, unit.Status, unit.Version)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: organization unit already exists", organization.ErrConflict)
		}
		return fmt.Errorf("create organization unit: %w", err)
	}
	return nil
}

func (s *mysqlOrganizationTxStore) UpdateUnit(ctx context.Context, unit organization.Unit, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_organization_units
SET parent_id = NULLIF(?, ''),
    name = ?,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND version = ?`,
		unit.ParentID, unit.Name, unit.ID, expectedVersion)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: organization unit conflicts with existing data", organization.ErrConflict)
		}
		return fmt.Errorf("update organization unit: %w", err)
	}
	return s.requireOrganizationMutation(ctx, result, unit.ID)
}

func (s *mysqlOrganizationTxStore) SetUnitStatus(ctx context.Context, unitID string, status organization.Status, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE identity_organization_units
SET status = ?,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ? AND version = ?`, status, unitID, expectedVersion)
	if err != nil {
		return fmt.Errorf("set organization unit status: %w", err)
	}
	return s.requireOrganizationMutation(ctx, result, unitID)
}

func (s *mysqlOrganizationTxStore) CountActiveChildren(ctx context.Context, unitID string) (int64, error) {
	var count int64
	err := s.tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM identity_organization_units
WHERE parent_id = ?
  AND unit_type = 'department'
  AND status = 'active'`, unitID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active organization children: %w", err)
	}
	return count, nil
}

func (s *mysqlOrganizationTxStore) CountActiveDoctors(ctx context.Context, departmentID string) (int64, error) {
	var count int64
	err := s.tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM identity_staff_profiles sp
JOIN identity_accounts a
  ON a.id = sp.account_id
 AND a.status = 'active'
JOIN identity_account_roles ar
  ON ar.account_id = a.id
JOIN identity_roles r
  ON r.id = ar.role_id
 AND r.code = 'department_doctor'
 AND r.status = 'active'
WHERE sp.department_id = ?
  AND sp.staff_status = 'active'`, departmentID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active department doctors: %w", err)
	}
	return count, nil
}

func (s *mysqlOrganizationTxStore) RecordChange(ctx context.Context, change organization.Change) error {
	var beforeData []byte
	var err error
	if change.Before != nil {
		beforeData, err = json.Marshal(change.Before)
		if err != nil {
			return fmt.Errorf("marshal organization before state: %w", err)
		}
	}
	afterData, err := json.Marshal(change.After)
	if err != nil {
		return fmt.Errorf("marshal organization after state: %w", err)
	}

	_, err = s.tx.ExecContext(ctx, `
INSERT INTO identity_organization_audit
    (id, operation_id, operator_account_id, unit_id, action, before_data, after_data, request_id)
VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`,
		uuid.NewString(), change.OperationID, change.OperatorAccountID, change.UnitID,
		change.Action, beforeData, afterData, change.RequestID)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: organization operation_id already exists", organization.ErrConflict)
		}
		return fmt.Errorf("insert organization audit: %w", err)
	}

	payload, err := json.Marshal(map[string]any{
		"operation_id":        change.OperationID,
		"operator_account_id": change.OperatorAccountID,
		"unit_id":             change.UnitID,
		"action":              change.Action,
		"before":              change.Before,
		"after":               change.After,
	})
	if err != nil {
		return fmt.Errorf("marshal organization event: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO identity_outbox_events
    (event_id, aggregate_id, event_type, schema_version, payload, occurred_at)
VALUES (?, ?, ?, 1, ?, ?)`,
		uuid.NewString(), change.UnitID, change.Action, payload, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("insert organization outbox: %w", err)
	}
	return nil
}

func (s *mysqlOrganizationTxStore) requireOrganizationMutation(ctx context.Context, result sql.Result, unitID string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read organization affected rows: %w", err)
	}
	if affected > 0 {
		return nil
	}

	var exists bool
	if err := s.tx.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM identity_organization_units
    WHERE id = ?
)`, unitID).Scan(&exists); err != nil {
		return fmt.Errorf("check organization unit after failed mutation: %w", err)
	}
	if !exists {
		return organization.ErrNotFound
	}
	return organization.ErrVersionConflict
}
