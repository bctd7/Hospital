package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	mysql "github.com/go-sql-driver/mysql"

	"hospital/service/guidance/rpc/internal/rules/precedence"
)

const ruleSelect = `
SELECT id, predecessor_item_id, predecessor_department_id, predecessor_item_name,
       successor_item_id, successor_department_id, successor_item_name,
       staff_reason, patient_message, created_by, create_operation_id,
       version, created_at, updated_at
FROM guidance_precedence_rules`

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type rowQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type ruleScanner interface{ Scan(...any) error }

func scanRule(scanner ruleScanner) (precedence.Rule, error) {
	var rule precedence.Rule
	err := scanner.Scan(
		&rule.RuleID,
		&rule.PredecessorItemID,
		&rule.PredecessorDepartmentID,
		&rule.PredecessorItemName,
		&rule.SuccessorItemID,
		&rule.SuccessorDepartmentID,
		&rule.SuccessorItemName,
		&rule.StaffReason,
		&rule.PatientMessage,
		&rule.CreatedBy,
		&rule.CreateOperationID,
		&rule.Version,
		&rule.CreatedAt,
		&rule.UpdatedAt,
	)
	return rule, err
}

func listRules(ctx context.Context, db queryer) ([]precedence.Rule, error) {
	rows, err := db.QueryContext(ctx, ruleSelect+" ORDER BY predecessor_item_id, successor_item_id")
	if err != nil {
		return nil, fmt.Errorf("list precedence rules: %w", err)
	}
	defer rows.Close()
	rules := make([]precedence.Rule, 0)
	for rows.Next() {
		rule, scanErr := scanRule(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan precedence rule: %w", scanErr)
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate precedence rules: %w", err)
	}
	return rules, nil
}

func getRule(row *sql.Row) (precedence.Rule, error) {
	rule, err := scanRule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return precedence.Rule{}, precedence.ErrNotFound
	}
	if err != nil {
		return precedence.Rule{}, fmt.Errorf("get precedence rule: %w", err)
	}
	return rule, nil
}

type txStore struct{ tx *sql.Tx }

func (s *txStore) LockGraph(ctx context.Context) error {
	var graph string
	if err := s.tx.QueryRowContext(ctx,
		"SELECT graph_name FROM guidance_precedence_graph_locks WHERE graph_name = 'examination_item_precedence' FOR UPDATE",
	).Scan(&graph); err != nil {
		return fmt.Errorf("lock precedence graph: %w", err)
	}
	return nil
}

func (s *txStore) ListAll(ctx context.Context) ([]precedence.Rule, error) {
	return listRules(ctx, s.tx)
}

func (s *txStore) GetByIDForUpdate(ctx context.Context, ruleID string) (precedence.Rule, error) {
	return getRule(s.tx.QueryRowContext(ctx, ruleSelect+" WHERE id = ? FOR UPDATE", ruleID))
}

func (s *txStore) GetByPair(ctx context.Context, predecessorItemID, successorItemID string) (precedence.Rule, error) {
	return getRule(s.tx.QueryRowContext(ctx,
		ruleSelect+" WHERE predecessor_item_id = ? AND successor_item_id = ?",
		predecessorItemID, successorItemID,
	))
}

func (s *txStore) GetByCreateOperationID(ctx context.Context, operationID string) (precedence.Rule, error) {
	return getRule(s.tx.QueryRowContext(ctx, ruleSelect+" WHERE create_operation_id = ?", operationID))
}

func (s *txStore) Insert(ctx context.Context, rule precedence.Rule) (precedence.Rule, error) {
	_, err := s.tx.ExecContext(ctx, `
INSERT INTO guidance_precedence_rules (
    id, predecessor_item_id, predecessor_department_id, predecessor_item_name,
    successor_item_id, successor_department_id, successor_item_name,
    staff_reason, patient_message, created_by, create_operation_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rule.RuleID, rule.PredecessorItemID, rule.PredecessorDepartmentID, rule.PredecessorItemName,
		rule.SuccessorItemID, rule.SuccessorDepartmentID, rule.SuccessorItemName,
		rule.StaffReason, rule.PatientMessage, rule.CreatedBy, rule.CreateOperationID,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return precedence.Rule{}, precedence.ErrConflict
		}
		return precedence.Rule{}, fmt.Errorf("insert precedence rule: %w", err)
	}
	return getRule(s.tx.QueryRowContext(ctx, ruleSelect+" WHERE id = ?", rule.RuleID))
}

func (s *txStore) UpdateText(ctx context.Context, ruleID, staffReason, patientMessage string, expectedVersion int64) (precedence.Rule, error) {
	result, err := s.tx.ExecContext(ctx, `
UPDATE guidance_precedence_rules
SET staff_reason = ?, patient_message = ?, version = version + 1
WHERE id = ? AND version = ?`, staffReason, patientMessage, ruleID, expectedVersion)
	if err != nil {
		return precedence.Rule{}, fmt.Errorf("update precedence rule: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return precedence.Rule{}, fmt.Errorf("read precedence update result: %w", err)
	}
	if changed == 0 {
		return precedence.Rule{}, precedence.ErrVersionConflict
	}
	return getRule(s.tx.QueryRowContext(ctx, ruleSelect+" WHERE id = ?", ruleID))
}

func (s *txStore) Delete(ctx context.Context, ruleID string, expectedVersion int64) error {
	result, err := s.tx.ExecContext(ctx,
		"DELETE FROM guidance_precedence_rules WHERE id = ? AND version = ?",
		ruleID, expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("delete precedence rule: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read precedence delete result: %w", err)
	}
	if changed == 0 {
		return precedence.ErrVersionConflict
	}
	return nil
}

var _ precedence.TxStore = (*txStore)(nil)
