package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"hospital/service/identity/rpc/internal/identityadmin"
)

var _ identityadmin.Store = (*Store)(nil)

const identityAdminAccountSelect = `
SELECT a.id,
       COALESCE(ap.nickname, ''),
       COALESCE(sp.display_name, ''),
       COALESCE(sp.avatar_url, ''),
       COALESCE(ph.phone_masked, ''),
       a.status,
       a.account_type,
       COALESCE(sp.staff_status, ''),
       COALESCE(sp.department_id, ''),
       COALESCE(ou.name, ''),
       a.management_version,
       COALESCE(ph.verification_status, ''),
       COALESCE(ph.verification_source, ''),
       COALESCE(sp.staff_no, ''),
       COALESCE(sp.description, ''),
       COALESCE(r.code, ''),
       a.authorization_version,
       a.created_at,
       a.updated_at
FROM identity_accounts a
LEFT JOIN identity_account_profiles ap ON ap.account_id = a.id
LEFT JOIN identity_staff_profiles sp ON sp.account_id = a.id
LEFT JOIN identity_organization_units ou ON ou.id = sp.department_id
LEFT JOIN identity_account_phones ph ON ph.account_id = a.id
LEFT JOIN identity_account_roles ar ON ar.account_id = a.id
LEFT JOIN identity_roles r ON r.id = ar.role_id`

func (s *Store) GetDisplayProfile(ctx context.Context, accountID string) (identityadmin.DisplayProfile, error) {
	var profile identityadmin.DisplayProfile
	err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(ap.nickname, ''), a.management_version
FROM identity_accounts a
LEFT JOIN identity_account_profiles ap ON ap.account_id = a.id
WHERE a.id = ?`, accountID).Scan(&profile.Nickname, &profile.ManagementVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return identityadmin.DisplayProfile{}, identityadmin.ErrNotFound
	}
	if err != nil {
		return identityadmin.DisplayProfile{}, fmt.Errorf("get account display profile: %w", err)
	}
	return profile, nil
}

func (s *Store) UpdateDisplayProfile(ctx context.Context, accountID, nickname string) (identityadmin.DisplayProfile, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return identityadmin.DisplayProfile{}, fmt.Errorf("begin display profile update: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
UPDATE identity_accounts
SET management_version = management_version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID)
	if err != nil {
		return identityadmin.DisplayProfile{}, fmt.Errorf("bump display profile version: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return identityadmin.DisplayProfile{}, fmt.Errorf("read display profile update result: %w", err)
	}
	if affected == 0 {
		return identityadmin.DisplayProfile{}, identityadmin.ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_account_profiles (account_id, nickname)
VALUES (?, NULLIF(?, ''))
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), updated_at = CURRENT_TIMESTAMP(3)`, accountID, nickname); err != nil {
		return identityadmin.DisplayProfile{}, fmt.Errorf("save account display profile: %w", err)
	}
	var profile identityadmin.DisplayProfile
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(ap.nickname, ''), a.management_version
FROM identity_accounts a
LEFT JOIN identity_account_profiles ap ON ap.account_id = a.id
WHERE a.id = ?`, accountID).Scan(&profile.Nickname, &profile.ManagementVersion); err != nil {
		return identityadmin.DisplayProfile{}, fmt.Errorf("read updated display profile: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return identityadmin.DisplayProfile{}, fmt.Errorf("commit display profile update: %w", err)
	}
	return profile, nil
}

func (s *Store) ListDoctors(ctx context.Context, departmentID string, page, pageSize int64) (identityadmin.DoctorPage, error) {
	var activeDepartment bool
	if err := s.db.QueryRowContext(ctx, `
SELECT EXISTS(
    SELECT 1 FROM identity_organization_units
    WHERE id = ? AND unit_type = 'department' AND status = 'active'
)`, departmentID).Scan(&activeDepartment); err != nil {
		return identityadmin.DoctorPage{}, fmt.Errorf("check public doctor department: %w", err)
	}
	if !activeDepartment {
		return identityadmin.DoctorPage{}, identityadmin.ErrNotFound
	}

	var total int64
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM identity_staff_profiles sp
JOIN identity_accounts a ON a.id = sp.account_id AND a.status = 'active'
JOIN identity_account_roles ar ON ar.account_id = a.id
JOIN identity_roles r ON r.id = ar.role_id AND r.code = 'department_doctor' AND r.status = 'active'
WHERE sp.department_id = ? AND sp.staff_status = 'active'`, departmentID).Scan(&total); err != nil {
		return identityadmin.DoctorPage{}, fmt.Errorf("count public doctors: %w", err)
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
		return identityadmin.DoctorPage{}, fmt.Errorf("list public doctors: %w", err)
	}
	defer rows.Close()
	items := make([]identityadmin.DoctorSummary, 0)
	for rows.Next() {
		var item identityadmin.DoctorSummary
		if err := rows.Scan(&item.AccountID, &item.DisplayName, &item.DepartmentID,
			&item.AvatarURL, &item.Description, &item.ManagementVersion); err != nil {
			return identityadmin.DoctorPage{}, fmt.Errorf("scan public doctor: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return identityadmin.DoctorPage{}, fmt.Errorf("iterate public doctors: %w", err)
	}
	return identityadmin.DoctorPage{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Store) GetAccount(ctx context.Context, accountID string) (identityadmin.Account, error) {
	return scanIdentityAdminAccount(s.db.QueryRowContext(ctx, identityAdminAccountSelect+" WHERE a.id = ?", accountID))
}

func (s *Store) FindAccountIDByPhone(ctx context.Context, fingerprint []byte) (string, error) {
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id FROM identity_account_phones WHERE phone_fingerprint = ?`, fingerprint).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", identityadmin.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find managed account by phone: %w", err)
	}
	return accountID, nil
}

func (s *Store) ListAccounts(ctx context.Context, filter identityadmin.AccountFilter) (identityadmin.AccountPage, error) {
	where := make([]string, 0, 4)
	args := make([]any, 0, 8)
	if filter.Nickname != "" {
		where = append(where, "(ap.nickname LIKE ? ESCAPE '\\\\' OR sp.display_name LIKE ? ESCAPE '\\\\')")
		prefix := escapedLikePrefix(filter.Nickname)
		args = append(args, prefix, prefix)
	}
	if filter.Status != "" {
		where = append(where, "a.status = ?")
		args = append(args, filter.Status)
	}
	if filter.DepartmentID != "" {
		where = append(where, "sp.department_id = ? AND sp.staff_status = 'active'")
		args = append(args, filter.DepartmentID)
	}
	switch filter.IdentityType {
	case identityadmin.IdentityTypeSuperAdmin:
		where = append(where, "r.code = 'super_admin'")
	case identityadmin.IdentityTypeDoctor:
		where = append(where, "r.code = 'department_doctor' AND sp.staff_status = 'active'")
	case identityadmin.IdentityTypePatient:
		where = append(where, "(r.code IS NULL OR r.code <> 'super_admin') AND (r.code IS NULL OR r.code <> 'department_doctor' OR sp.staff_status <> 'active')")
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	var total int64
	countQuery := `
SELECT COUNT(*)
FROM identity_accounts a
LEFT JOIN identity_account_profiles ap ON ap.account_id = a.id
LEFT JOIN identity_staff_profiles sp ON sp.account_id = a.id
LEFT JOIN identity_account_roles ar ON ar.account_id = a.id
LEFT JOIN identity_roles r ON r.id = ar.role_id` + whereSQL
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return identityadmin.AccountPage{}, fmt.Errorf("count managed accounts: %w", err)
	}

	queryArgs := append(append([]any{}, args...), filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.db.QueryContext(ctx, identityAdminAccountSelect+whereSQL+`
ORDER BY COALESCE(NULLIF(sp.display_name, ''), NULLIF(ap.nickname, ''), a.id), a.id
LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return identityadmin.AccountPage{}, fmt.Errorf("list managed accounts: %w", err)
	}
	defer rows.Close()
	items := make([]identityadmin.Account, 0)
	for rows.Next() {
		account, err := scanIdentityAdminAccount(rows)
		if err != nil {
			return identityadmin.AccountPage{}, err
		}
		items = append(items, account)
	}
	if err := rows.Err(); err != nil {
		return identityadmin.AccountPage{}, fmt.Errorf("iterate managed accounts: %w", err)
	}
	return identityadmin.AccountPage{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, nil
}

type identityAdminScanner interface {
	Scan(dest ...any) error
}

func scanIdentityAdminAccount(scanner identityAdminScanner) (identityadmin.Account, error) {
	var account identityadmin.Account
	var role string
	err := scanner.Scan(
		&account.ID, &account.Nickname, &account.DisplayName, &account.AvatarURL,
		&account.MaskedPhone, &account.AccountStatus, &account.AccountType,
		&account.StaffStatus, &account.DepartmentID, &account.DepartmentName,
		&account.ManagementVersion, &account.PhoneVerificationStatus,
		&account.PhoneVerificationSource, &account.StaffNo, &account.Description,
		&role, &account.AuthorizationVersion, &account.CreatedAt, &account.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return identityadmin.Account{}, identityadmin.ErrNotFound
	}
	if err != nil {
		return identityadmin.Account{}, fmt.Errorf("scan managed account: %w", err)
	}
	if role != "" {
		account.Roles = []string{role}
	} else {
		account.Roles = []string{}
	}
	return account, nil
}

func escapedLikePrefix(value string) string {
	value = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
	return value + "%"
}
