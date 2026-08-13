package mysqlstore

import (
	"context"
	"fmt"

	"hospital/service/identity/rpc/internal/account"
)

func (s *Store) ListDoctors(ctx context.Context, departmentID string, page, pageSize int64) (account.DoctorPage, error) {
	var activeDepartment bool
	if err := s.db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1 FROM identity_organization_units
    WHERE id = ? AND unit_type = 'department' AND status = 'active'
)`, departmentID).Scan(&activeDepartment); err != nil {
		return account.DoctorPage{}, fmt.Errorf("check public doctor department: %w", err)
	}
	if !activeDepartment {
		return account.DoctorPage{}, account.ErrNotFound
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM identity_staff_profiles sp
JOIN identity_accounts a ON a.id = sp.account_id AND a.status = 'active'
JOIN identity_account_roles ar ON ar.account_id = a.id
JOIN identity_roles r ON r.id = ar.role_id AND r.code = 'department_doctor' AND r.status = 'active'
WHERE sp.department_id = ? AND sp.staff_status = 'active'`, departmentID).Scan(&total); err != nil {
		return account.DoctorPage{}, fmt.Errorf("count public doctors: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT sp.account_id, sp.display_name, sp.department_id,
       COALESCE(sp.avatar_url, ''), COALESCE(sp.description, ''), a.management_version
FROM identity_staff_profiles sp
JOIN identity_accounts a ON a.id = sp.account_id AND a.status = 'active'
JOIN identity_account_roles ar ON ar.account_id = a.id
JOIN identity_roles r ON r.id = ar.role_id AND r.code = 'department_doctor' AND r.status = 'active'
WHERE sp.department_id = ? AND sp.staff_status = 'active'
ORDER BY sp.display_name, COALESCE(sp.staff_no, ''), sp.account_id
LIMIT ? OFFSET ?`, departmentID, pageSize, (page-1)*pageSize)
	if err != nil {
		return account.DoctorPage{}, fmt.Errorf("list public doctors: %w", err)
	}
	defer rows.Close()
	items := make([]account.DoctorSummary, 0)
	for rows.Next() {
		var item account.DoctorSummary
		if err := rows.Scan(&item.AccountID, &item.DisplayName, &item.DepartmentID,
			&item.AvatarURL, &item.Description, &item.ManagementVersion); err != nil {
			return account.DoctorPage{}, fmt.Errorf("scan public doctor: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return account.DoctorPage{}, fmt.Errorf("iterate public doctors: %w", err)
	}
	return account.DoctorPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}
