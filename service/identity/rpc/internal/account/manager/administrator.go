package manager

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/identity/rpc/internal/account"
)

func (m *Manager) ListAccounts(ctx context.Context, operator authn.Principal, filter account.AccountFilter) (account.AccountPage, error) {
	if !canReadAccounts(operator) {
		return account.AccountPage{}, account.ErrForbidden
	}
	page, pageSize, err := normalizedPage(filter.Page, filter.PageSize)
	if err != nil {
		return account.AccountPage{}, err
	}
	filter.Page, filter.PageSize = page, pageSize
	filter.Nickname, err = normalizedText(filter.Nickname, "nickname", maxDisplayNameRunes, true)
	if err != nil {
		return account.AccountPage{}, err
	}
	filter.IdentityType = strings.TrimSpace(filter.IdentityType)
	if filter.IdentityType != "" && filter.IdentityType != account.IdentityTypePatient &&
		filter.IdentityType != account.IdentityTypeDoctor && filter.IdentityType != account.IdentityTypeSuperAdmin {
		return account.AccountPage{}, fmt.Errorf("%w: unsupported identity_type %q", account.ErrInvalid, filter.IdentityType)
	}
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != "" && filter.Status != authn.AccountStatusActive && filter.Status != authn.AccountStatusDisabled {
		return account.AccountPage{}, fmt.Errorf("%w: unsupported status %q", account.ErrInvalid, filter.Status)
	}
	if strings.TrimSpace(filter.DepartmentID) != "" {
		filter.DepartmentID, err = normalizedUUID(filter.DepartmentID, "department_id")
		if err != nil {
			return account.AccountPage{}, err
		}
	}
	return m.store.ListAccounts(ctx, filter)
}

func (m *Manager) GetAccount(ctx context.Context, operator authn.Principal, accountID string) (account.Account, []string, error) {
	if !canReadAccounts(operator) {
		return account.Account{}, nil, account.ErrForbidden
	}
	accountID, err := normalizedUUID(accountID, "account_id")
	if err != nil {
		return account.Account{}, nil, err
	}
	managedAccount, err := m.store.GetAccount(ctx, accountID)
	if err != nil {
		return account.Account{}, nil, err
	}
	return managedAccount, availableActions(operator, managedAccount), nil
}

func (m *Manager) SearchByPhone(ctx context.Context, operator authn.Principal, phone string) (account.Account, string, string, string, error) {
	if !canReadAccounts(operator) {
		return account.Account{}, "", "", "", account.ErrForbidden
	}
	normalized, err := normalizePhone(phone)
	if err != nil {
		return account.Account{}, "", "", "", err
	}
	fingerprint := hmac.New(sha256.New, m.phoneKey)
	_, _ = fingerprint.Write([]byte(normalized))
	accountID, err := m.store.FindAccountIDByPhone(ctx, fingerprint.Sum(nil))
	if err != nil {
		return account.Account{}, "", "", "", err
	}
	managedAccount, err := m.store.GetAccount(ctx, accountID)
	if err != nil {
		return account.Account{}, "", "", "", err
	}
	return managedAccount, managedAccount.MaskedPhone, managedAccount.PhoneVerificationStatus, managedAccount.PhoneVerificationSource, nil
}

func (m *Manager) SetAccountEnabled(ctx context.Context, operator authn.Principal, accountID string, enabled bool, expectedVersion int64, operationID, requestID string) (account.Account, []string, error) {
	status := authn.AccountStatusDisabled
	action := account.ActionDisableAccount
	if enabled {
		status = authn.AccountStatusActive
		action = account.ActionEnableAccount
	}
	return m.mutate(ctx, operator, accountID, expectedVersion, operationID, requestID,
		action, contractauthz.PermissionIdentityAccountManage,
		func(tx account.TxStore, before account.Account) error {
			if before.AccountStatus == status {
				return fmt.Errorf("%w: account already has requested status", account.ErrInvalidState)
			}
			return tx.SetManagedAccountStatus(ctx, before.ID, status, expectedVersion)
		})
}

func availableActions(operator authn.Principal, target account.Account) []string {
	if operator.AccountID == target.ID || target.IdentityType() == account.IdentityTypeSuperAdmin {
		return []string{}
	}
	if target.AccountStatus == authn.AccountStatusDisabled {
		if operator.HasPermission(contractauthz.PermissionIdentityAccountManage) {
			return []string{"enable_account"}
		}
		return []string{}
	}
	actions := make([]string, 0, 4)
	if operator.HasPermission(contractauthz.PermissionIdentityAuthorizationManage) {
		if target.IdentityType() == account.IdentityTypeDoctor {
			actions = append(actions, "edit_doctor", "change_department", "revoke_doctor")
		} else {
			actions = append(actions, "promote_doctor")
		}
	}
	if operator.HasPermission(contractauthz.PermissionIdentityAccountManage) {
		actions = append(actions, "disable_account")
	}
	return actions
}

func canReadAccounts(operator authn.Principal) bool {
	return operator.HasPermission(contractauthz.PermissionIdentityAuthorizationManage) ||
		operator.HasPermission(contractauthz.PermissionIdentityAccountManage)
}
