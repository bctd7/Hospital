package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"hospital/service/identity/rpc/internal/organization"
)

var _ organization.Store = (*Store)(nil)

const organizationUnitSelect = `
SELECT
    ou.id,
    COALESCE(ou.parent_id, ''),
    ou.unit_type,
    ou.code,
    ou.name,
    ou.status,
    CASE
        WHEN ou.unit_type = 'campus' THEN (
            SELECT COUNT(*)
            FROM identity_organization_units child
            WHERE child.parent_id = ou.id
              AND child.unit_type = 'department'
              AND child.status = 'active'
        )
        ELSE 0
    END AS child_count,
    CASE
        WHEN ou.unit_type = 'department' THEN (
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
            WHERE sp.department_id = ou.id
        )
        ELSE 0
    END AS doctor_count,
    ou.version
FROM identity_organization_units ou`

func (s *Store) GetUnit(ctx context.Context, unitID string) (organization.Unit, error) {
	unit, err := scanOrganizationUnit(s.db.QueryRowContext(ctx, organizationUnitSelect+" WHERE ou.id = ?", unitID))
	if errors.Is(err, sql.ErrNoRows) {
		return organization.Unit{}, organization.ErrNotFound
	}
	if err != nil {
		return organization.Unit{}, fmt.Errorf("get organization unit: %w", err)
	}
	return unit, nil
}

func (s *Store) ListUnits(ctx context.Context, filter organization.ListFilter) ([]organization.Unit, error) {
	query := organizationUnitSelect + " WHERE ou.unit_type = ?"
	args := []any{filter.Type}
	if filter.ParentID != nil {
		query += " AND ou.parent_id = ?"
		args = append(args, *filter.ParentID)
	}
	if filter.Status != nil {
		query += " AND ou.status = ?"
		args = append(args, *filter.Status)
	}
	query += " ORDER BY ou.code, ou.id"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list organization units: %w", err)
	}
	defer rows.Close()

	units := make([]organization.Unit, 0)
	for rows.Next() {
		unit, err := scanOrganizationUnit(rows)
		if err != nil {
			return nil, fmt.Errorf("scan organization unit list: %w", err)
		}
		units = append(units, unit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organization unit list: %w", err)
	}
	return units, nil
}

func (s *Store) WithinOrganizationTransaction(ctx context.Context, fn func(organization.TxStore) error) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("begin organization transaction: %w", err)
	}
	store := &mysqlOrganizationTxStore{tx: tx}
	if err := fn(store); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return errors.Join(err, fmt.Errorf("rollback organization transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit organization transaction: %w", err)
	}
	return nil
}

type organizationUnitScanner interface {
	Scan(dest ...any) error
}

func scanOrganizationUnit(scanner organizationUnitScanner) (organization.Unit, error) {
	var unit organization.Unit
	err := scanner.Scan(
		&unit.ID,
		&unit.ParentID,
		&unit.Type,
		&unit.Code,
		&unit.Name,
		&unit.Status,
		&unit.ChildCount,
		&unit.DoctorCount,
		&unit.Version,
	)
	return unit, err
}
