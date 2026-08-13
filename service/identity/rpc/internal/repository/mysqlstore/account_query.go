package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"hospital/service/identity/rpc/internal/account"
)

var _ account.Store = (*Store)(nil)

const managedAccountSelect = `
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

func (s *Store) GetAccount(ctx context.Context, accountID string) (account.Account, error) {
	return scanManagedAccount(s.db.QueryRowContext(ctx, managedAccountSelect+" WHERE a.id = ?", accountID))
}

func (s *Store) FindAccountIDByPhone(ctx context.Context, fingerprint []byte) (string, error) {
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id FROM identity_account_phones WHERE phone_fingerprint = ?`, fingerprint).Scan(&accountID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", account.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find managed account by phone: %w", err)
	}
	return accountID, nil
}

func (s *Store) ListAccounts(ctx context.Context, filter account.AccountFilter) (account.AccountPage, error) {
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
	case account.IdentityTypeSuperAdmin:
		where = append(where, "r.code = 'super_admin'")
	case account.IdentityTypeDoctor:
		where = append(where, "r.code = 'department_doctor' AND sp.staff_status = 'active'")
	case account.IdentityTypePatient:
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
		return account.AccountPage{}, fmt.Errorf("count managed accounts: %w", err)
	}

	queryArgs := append(append([]any{}, args...), filter.PageSize, (filter.Page-1)*filter.PageSize)
	rows, err := s.db.QueryContext(ctx, managedAccountSelect+whereSQL+`
ORDER BY COALESCE(NULLIF(sp.display_name, ''), NULLIF(ap.nickname, ''), a.id), a.id
LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return account.AccountPage{}, fmt.Errorf("list managed accounts: %w", err)
	}
	defer rows.Close()
	items := make([]account.Account, 0)
	for rows.Next() {
		managedAccount, err := scanManagedAccount(rows)
		if err != nil {
			return account.AccountPage{}, err
		}
		items = append(items, managedAccount)
	}
	if err := rows.Err(); err != nil {
		return account.AccountPage{}, fmt.Errorf("iterate managed accounts: %w", err)
	}
	return account.AccountPage{Items: items, Page: filter.Page, PageSize: filter.PageSize, Total: total}, nil
}

type accountScanner interface {
	Scan(dest ...any) error
}

func scanManagedAccount(scanner accountScanner) (account.Account, error) {
	var managedAccount account.Account
	var role string
	err := scanner.Scan(
		&managedAccount.ID, &managedAccount.Nickname, &managedAccount.DisplayName, &managedAccount.AvatarURL,
		&managedAccount.MaskedPhone, &managedAccount.AccountStatus, &managedAccount.AccountType,
		&managedAccount.StaffStatus, &managedAccount.DepartmentID, &managedAccount.DepartmentName,
		&managedAccount.ManagementVersion, &managedAccount.PhoneVerificationStatus,
		&managedAccount.PhoneVerificationSource, &managedAccount.StaffNo, &managedAccount.Description,
		&role, &managedAccount.AuthorizationVersion, &managedAccount.CreatedAt, &managedAccount.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return account.Account{}, account.ErrNotFound
	}
	if err != nil {
		return account.Account{}, fmt.Errorf("scan managed account: %w", err)
	}
	if role != "" {
		managedAccount.Roles = []string{role}
	} else {
		managedAccount.Roles = []string{}
	}
	return managedAccount, nil
}

func escapedLikePrefix(value string) string {
	value = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
	return value + "%"
}
