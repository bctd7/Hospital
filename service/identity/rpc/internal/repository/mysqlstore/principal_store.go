package mysqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/account"
	"hospital/service/identity/rpc/internal/session"
)

var _ session.PrincipalStore = (*Store)(nil)

func (s *Store) GetPrincipal(ctx context.Context, accountID string) (authn.Principal, error) {
	return readPrincipal(ctx, s.db, accountID)
}

type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func readPrincipal(ctx context.Context, db queryer, accountID string) (authn.Principal, error) {
	query := `
SELECT
    a.id,
    a.account_type,
    a.status,
    a.authorization_version,
    COALESCE(sp.department_id, ''),
    COALESCE(r.code, ''),
    COALESCE(p.code, '')
FROM identity_accounts a
LEFT JOIN identity_staff_profiles sp ON sp.account_id = a.id
LEFT JOIN identity_account_roles ar ON ar.account_id = a.id
LEFT JOIN identity_roles r ON r.id = ar.role_id AND r.status = 'active'
LEFT JOIN identity_role_permissions rp ON rp.role_id = r.id
LEFT JOIN identity_permissions p ON p.id = rp.permission_id AND p.status = 'active'
WHERE a.id = ?`
	rows, err := db.QueryContext(ctx, query, accountID)
	if err != nil {
		return authn.Principal{}, fmt.Errorf("query identity principal: %w", err)
	}
	defer rows.Close()

	var principal authn.Principal
	roles := make(map[string]struct{})
	permissions := make(map[string]struct{})
	found := false
	for rows.Next() {
		var role, permission string
		if err := rows.Scan(
			&principal.AccountID, &principal.AccountType, &principal.Status,
			&principal.AuthorizationVersion, &principal.DepartmentID, &role, &permission,
		); err != nil {
			return authn.Principal{}, fmt.Errorf("scan identity principal: %w", err)
		}
		found = true
		if role != "" {
			roles[role] = struct{}{}
		}
		if permission != "" {
			permissions[permission] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return authn.Principal{}, fmt.Errorf("iterate identity principal: %w", err)
	}
	if !found {
		return authn.Principal{}, account.ErrNotFound
	}
	principal.Roles = sortedKeys(roles)
	principal.Permissions = sortedKeys(permissions)
	return principal, nil
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
