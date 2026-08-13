package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"hospital/service/appointment/rpc/internal/catalog"
)

var _ catalog.TxStore = (*catalogTxStore)(nil)

type catalogTxStore struct {
	tx *sql.Tx
}

func (s *catalogTxStore) FindOperation(ctx context.Context, operationID string) (catalog.Operation, bool, error) {
	var operation catalog.Operation
	var resultData []byte
	err := s.tx.QueryRowContext(ctx, `
SELECT operator_account_id, item_id, action, request_fingerprint, result_data
FROM appointment_examination_item_operations
WHERE operation_id = ?`, operationID).Scan(
		&operation.OperatorAccountID,
		&operation.ItemID,
		&operation.Action,
		&operation.RequestFingerprint,
		&resultData,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.Operation{}, false, nil
	}
	if err != nil {
		return catalog.Operation{}, false, fmt.Errorf("find examination item operation: %w", err)
	}
	if err := json.Unmarshal(resultData, &operation.Result); err != nil {
		return catalog.Operation{}, false, fmt.Errorf("decode examination item operation result: %w", err)
	}
	return operation, true, nil
}

func (s *catalogTxStore) GetItemForUpdate(ctx context.Context, itemID string) (catalog.ExaminationItem, error) {
	item, err := scanExaminationItem(s.tx.QueryRowContext(ctx, examinationItemSelect+" WHERE id = ? FOR UPDATE", itemID))
	if errors.Is(err, sql.ErrNoRows) {
		return catalog.ExaminationItem{}, catalog.ErrNotFound
	}
	if err != nil {
		return catalog.ExaminationItem{}, fmt.Errorf("lock examination item: %w", err)
	}
	return item, nil
}

func (s *catalogTxStore) CreateItem(ctx context.Context, item catalog.ExaminationItem) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_examination_items
    (id, owner_department_id, name, description, status, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ItemID,
		item.OwnerDepartmentID,
		item.Name,
		item.Description,
		item.Status,
		item.Version,
		item.CreatedAt,
		item.UpdatedAt,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: examination item already exists", catalog.ErrConflict)
		}
		return fmt.Errorf("create examination item: %w", err)
	}
	return nil
}

func (s *catalogTxStore) UpdateItem(ctx context.Context, item catalog.ExaminationItem, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_examination_items
SET name = ?,
    description = ?,
    version = version + 1,
    updated_at = ?
WHERE id = ? AND version = ?`,
		item.Name,
		item.Description,
		item.UpdatedAt,
		item.ItemID,
		expectedVersion,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: examination item name already exists in department", catalog.ErrConflict)
		}
		return fmt.Errorf("update examination item: %w", err)
	}
	return s.requireCatalogMutation(ctx, result, item.ItemID)
}

func (s *catalogTxStore) SetItemStatus(ctx context.Context, item catalog.ExaminationItem, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_examination_items
SET status = ?,
    version = version + 1,
    updated_at = ?
WHERE id = ? AND version = ?`,
		item.Status,
		item.UpdatedAt,
		item.ItemID,
		expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("set examination item status: %w", err)
	}
	return s.requireCatalogMutation(ctx, result, item.ItemID)
}

func (s *catalogTxStore) RecordChange(ctx context.Context, change catalog.Change) error {
	resultData, err := json.Marshal(change.After)
	if err != nil {
		return fmt.Errorf("marshal examination item operation result: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO appointment_examination_item_operations
    (operation_id, operator_account_id, item_id, action, request_fingerprint, result_data)
VALUES (?, ?, ?, ?, ?, ?)`,
		change.OperationID,
		change.OperatorAccountID,
		change.ItemID,
		change.Action,
		change.RequestFingerprint,
		resultData,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: examination catalog operation_id already exists", catalog.ErrConflict)
		}
		return fmt.Errorf("record examination item operation: %w", err)
	}

	descriptionChanged := change.Before == nil || change.Before.Description != change.After.Description
	var beforeData []byte
	if change.Before != nil {
		beforeData, err = json.Marshal(newAuditState(*change.Before, false))
		if err != nil {
			return fmt.Errorf("marshal examination item before audit state: %w", err)
		}
	}
	afterData, err := json.Marshal(newAuditState(change.After, descriptionChanged))
	if err != nil {
		return fmt.Errorf("marshal examination item after audit state: %w", err)
	}
	_, err = s.tx.ExecContext(ctx, `
INSERT INTO appointment_examination_item_audit
    (id, operation_id, operator_account_id, item_id, action, before_data, after_data, request_id)
VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''))`,
		uuid.NewString(),
		change.OperationID,
		change.OperatorAccountID,
		change.ItemID,
		change.Action,
		beforeData,
		afterData,
		change.RequestID,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return fmt.Errorf("%w: examination catalog audit already exists", catalog.ErrConflict)
		}
		return fmt.Errorf("record examination item audit: %w", err)
	}
	return nil
}

func (s *catalogTxStore) requireCatalogMutation(ctx context.Context, result sql.Result, itemID string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read examination item affected rows: %w", err)
	}
	if affected > 0 {
		return nil
	}

	var exists bool
	if err := s.tx.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1
    FROM appointment_examination_items
    WHERE id = ?
)`, itemID).Scan(&exists); err != nil {
		return fmt.Errorf("check examination item after failed mutation: %w", err)
	}
	if !exists {
		return catalog.ErrNotFound
	}
	return catalog.ErrVersionConflict
}

type examinationItemAuditState struct {
	ItemID             string         `json:"item_id"`
	OwnerDepartmentID  string         `json:"owner_department_id"`
	Name               string         `json:"name"`
	Status             catalog.Status `json:"status"`
	Version            int64          `json:"version"`
	DescriptionChanged bool           `json:"description_changed"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

func newAuditState(item catalog.ExaminationItem, descriptionChanged bool) examinationItemAuditState {
	return examinationItemAuditState{
		ItemID:             item.ItemID,
		OwnerDepartmentID:  item.OwnerDepartmentID,
		Name:               item.Name,
		Status:             item.Status,
		Version:            item.Version,
		DescriptionChanged: descriptionChanged,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func isDuplicateEntry(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
