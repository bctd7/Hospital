package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
)

var _ appointmentmanager.ProjectConfigurationTxStore = (*projectConfigurationTxStore)(nil)

type projectConfigurationTxStore struct{ *projectTxStore }

const projectConfigurationTransactionSelect = `
SELECT transaction_id, operation_id, operator_account_id, action, item_id,
       owner_department_id, item_name, description_snapshot, estimated_duration_minutes, expected_version,
       state, request_fingerprint, result_version, created_at, updated_at
FROM appointment_examination_item_configuration_transactions`

func scanProjectConfigurationTransaction(scanner interface{ Scan(...any) error }) (appointmentmanager.ProjectConfigurationTransaction, error) {
	var value appointmentmanager.ProjectConfigurationTransaction
	err := scanner.Scan(
		&value.TransactionID, &value.OperationID, &value.OperatorAccountID, &value.Action, &value.ItemID,
		&value.OwnerDepartmentID, &value.ItemName, &value.Description, &value.EstimatedDurationMinutes, &value.ExpectedVersion,
		&value.State, &value.RequestFingerprint, &value.ResultVersion, &value.CreatedAt, &value.UpdatedAt,
	)
	return value, err
}

func (s *projectConfigurationTxStore) GetConfigurationTransactionForUpdate(ctx context.Context, transactionID string) (appointmentmanager.ProjectConfigurationTransaction, bool, error) {
	value, err := scanProjectConfigurationTransaction(s.tx.QueryRowContext(ctx, projectConfigurationTransactionSelect+" WHERE transaction_id = ? FOR UPDATE", transactionID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ProjectConfigurationTransaction{}, false, nil
	}
	if err != nil {
		return appointmentmanager.ProjectConfigurationTransaction{}, false, fmt.Errorf("get project configuration transaction: %w", err)
	}
	return value, true, nil
}

func (s *projectConfigurationTxStore) FindConfigurationTransactionByOperation(ctx context.Context, operationID string) (appointmentmanager.ProjectConfigurationTransaction, bool, error) {
	value, err := scanProjectConfigurationTransaction(s.tx.QueryRowContext(ctx, projectConfigurationTransactionSelect+" WHERE operation_id = ? FOR UPDATE", operationID))
	if errors.Is(err, sql.ErrNoRows) {
		return appointmentmanager.ProjectConfigurationTransaction{}, false, nil
	}
	if err != nil {
		return appointmentmanager.ProjectConfigurationTransaction{}, false, fmt.Errorf("find project configuration transaction by operation: %w", err)
	}
	return value, true, nil
}

func (s *projectConfigurationTxStore) CreateConfigurationTransaction(ctx context.Context, value appointmentmanager.ProjectConfigurationTransaction) error {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO appointment_examination_item_configuration_transactions
    (transaction_id, operation_id, operator_account_id, action, item_id,
     owner_department_id, item_name, description_snapshot, estimated_duration_minutes, expected_version,
     state, request_fingerprint, result_version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.TransactionID, value.OperationID, value.OperatorAccountID, value.Action, value.ItemID,
		value.OwnerDepartmentID, value.ItemName, value.Description, value.EstimatedDurationMinutes, value.ExpectedVersion,
		value.State, value.RequestFingerprint, value.ResultVersion, value.CreatedAt, value.UpdatedAt,
	)
	if err != nil {
		if isDuplicateEntry(err) {
			return appointmentmanager.ErrConflict
		}
		return fmt.Errorf("create project configuration transaction: %w", err)
	}
	return nil
}

func (s *projectConfigurationTxStore) SetConfigurationTransactionState(ctx context.Context, transactionID, state string, resultVersion int64, updatedAt time.Time) error {
	result, err := s.tx.ExecContext(ctx, `
UPDATE appointment_examination_item_configuration_transactions
SET state = ?, result_version = ?, updated_at = ?
WHERE transaction_id = ?`, state, resultVersion, updatedAt, transactionID)
	if err != nil {
		return fmt.Errorf("set project configuration transaction state: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read project configuration transaction affected rows: %w", err)
	}
	if affected != 1 {
		return appointmentmanager.ErrNotFound
	}
	return nil
}
