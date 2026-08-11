package identityadmin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

func (m *Manager) ListAccounts(ctx context.Context, operator authn.Principal, filter AccountFilter) (AccountPage, error) {
	if !canReadAccounts(operator) {
		return AccountPage{}, ErrForbidden
	}
	page, pageSize, err := normalizedPage(filter.Page, filter.PageSize)
	if err != nil {
		return AccountPage{}, err
	}
	filter.Page, filter.PageSize = page, pageSize
	filter.Nickname, err = normalizedText(filter.Nickname, "nickname", maxDisplayNameRunes, true)
	if err != nil {
		return AccountPage{}, err
	}
	filter.IdentityType = strings.TrimSpace(filter.IdentityType)
	if filter.IdentityType != "" && filter.IdentityType != IdentityTypePatient &&
		filter.IdentityType != IdentityTypeDoctor && filter.IdentityType != IdentityTypeSuperAdmin {
		return AccountPage{}, fmt.Errorf("%w: unsupported identity_type %q", ErrInvalid, filter.IdentityType)
	}
	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != "" && filter.Status != authn.AccountStatusActive && filter.Status != authn.AccountStatusDisabled {
		return AccountPage{}, fmt.Errorf("%w: unsupported status %q", ErrInvalid, filter.Status)
	}
	if strings.TrimSpace(filter.DepartmentID) != "" {
		filter.DepartmentID, err = normalizedUUID(filter.DepartmentID, "department_id")
		if err != nil {
			return AccountPage{}, err
		}
	}
	return m.store.ListAccounts(ctx, filter)
}

func (m *Manager) GetAccount(ctx context.Context, operator authn.Principal, accountID string) (Account, []string, error) {
	if !canReadAccounts(operator) {
		return Account{}, nil, ErrForbidden
	}
	accountID, err := normalizedUUID(accountID, "account_id")
	if err != nil {
		return Account{}, nil, err
	}
	account, err := m.store.GetAccount(ctx, accountID)
	if err != nil {
		return Account{}, nil, err
	}
	return account, availableActions(operator, account), nil
}

func (m *Manager) SearchByPhone(ctx context.Context, operator authn.Principal, phone string) (Account, string, string, string, error) {
	if !canReadAccounts(operator) {
		return Account{}, "", "", "", ErrForbidden
	}
	normalized, err := normalizePhone(phone)
	if err != nil {
		return Account{}, "", "", "", err
	}
	fingerprint := hmac.New(sha256.New, m.phoneKey)
	_, _ = fingerprint.Write([]byte(normalized))
	accountID, err := m.store.FindAccountIDByPhone(ctx, fingerprint.Sum(nil))
	if err != nil {
		return Account{}, "", "", "", err
	}
	account, err := m.store.GetAccount(ctx, accountID)
	if err != nil {
		return Account{}, "", "", "", err
	}
	return account, account.MaskedPhone, account.PhoneVerificationStatus, account.PhoneVerificationSource, nil
}

func (m *Manager) SetAccountEnabled(ctx context.Context, operator authn.Principal, accountID string, enabled bool, expectedVersion int64, operationID, requestID string) (Account, []string, error) {
	status := authn.AccountStatusDisabled
	action := ActionDisableAccount
	if enabled {
		status = authn.AccountStatusActive
		action = ActionEnableAccount
	}
	return m.mutate(ctx, operator, accountID, expectedVersion, operationID, requestID,
		action, contractauthz.PermissionIdentityAccountManage,
		func(tx TxStore, before Account) error {
			if before.AccountStatus == status {
				return fmt.Errorf("%w: account already has requested status", ErrInvalidState)
			}
			return tx.SetManagedAccountStatus(ctx, before.ID, status, expectedVersion)
		})
}

func availableActions(operator authn.Principal, target Account) []string {
	if operator.AccountID == target.ID || target.IdentityType() == IdentityTypeSuperAdmin {
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
		if target.IdentityType() == IdentityTypeDoctor {
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
