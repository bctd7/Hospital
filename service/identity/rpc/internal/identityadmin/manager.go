package identityadmin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

const (
	maxPageSize         = 100
	maxNicknameRunes    = 64
	maxDisplayNameRunes = 128
	maxStaffNoRunes     = 64
	maxDescriptionRunes = 512
	maxAvatarURLBytes   = 2048
	maxRequestIDBytes   = 64
)

var mainlandPhone = regexp.MustCompile(`^1[3-9][0-9]{9}$`)

type AuthorizationVersionPublisher interface {
	SetAuthorizationVersion(ctx context.Context, accountID string, version int64) error
}

type Manager struct {
	store    Store
	versions AuthorizationVersionPublisher
	phoneKey []byte
}

func NewManager(store Store, versions AuthorizationVersionPublisher, phoneKey []byte) (*Manager, error) {
	if store == nil || versions == nil || len(phoneKey) < 32 {
		return nil, errors.New("identity admin manager dependencies are required")
	}
	return &Manager{store: store, versions: versions, phoneKey: append([]byte(nil), phoneKey...)}, nil
}

func (m *Manager) GetDisplayProfile(ctx context.Context, operator authn.Principal) (DisplayProfile, error) {
	if err := validUUID(operator.AccountID, "account_id"); err != nil {
		return DisplayProfile{}, ErrForbidden
	}
	return m.store.GetDisplayProfile(ctx, operator.AccountID)
}

func (m *Manager) UpdateDisplayProfile(ctx context.Context, operator authn.Principal, nickname string) (DisplayProfile, error) {
	if err := validUUID(operator.AccountID, "account_id"); err != nil {
		return DisplayProfile{}, ErrForbidden
	}
	nickname, err := normalizedText(nickname, "nickname", maxNicknameRunes, true)
	if err != nil {
		return DisplayProfile{}, err
	}
	return m.store.UpdateDisplayProfile(ctx, operator.AccountID, nickname)
}

func (m *Manager) ListDoctors(ctx context.Context, departmentID string, page, pageSize int64) (DoctorPage, error) {
	departmentID, err := normalizedUUID(departmentID, "department_id")
	if err != nil {
		return DoctorPage{}, err
	}
	page, pageSize, err = normalizedPage(page, pageSize)
	if err != nil {
		return DoctorPage{}, err
	}
	return m.store.ListDoctors(ctx, departmentID, page, pageSize)
}

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

func (m *Manager) PromoteDoctor(
	ctx context.Context,
	operator authn.Principal,
	accountID, departmentID string,
	profile DoctorProfileInput,
	expectedVersion int64,
	offlineVerified bool,
	operationID, requestID string,
) (Account, []string, error) {
	if !offlineVerified {
		return Account{}, nil, fmt.Errorf("%w: offline identity verification is required", ErrInvalid)
	}
	departmentID, err := normalizedUUID(departmentID, "department_id")
	if err != nil {
		return Account{}, nil, err
	}
	profile, err = normalizedDoctorProfile(profile)
	if err != nil {
		return Account{}, nil, err
	}
	return m.mutate(ctx, operator, accountID, expectedVersion, operationID, requestID,
		ActionPromoteDoctor, contractauthz.PermissionIdentityAuthorizationManage,
		func(tx TxStore, before Account) error {
			if before.AccountStatus != authn.AccountStatusActive || before.IdentityType() != IdentityTypePatient {
				return fmt.Errorf("%w: only active patient accounts can be promoted", ErrInvalidState)
			}
			return tx.PromoteDoctor(ctx, before.ID, departmentID, profile, expectedVersion, true)
		})
}

func (m *Manager) UpdateDoctor(
	ctx context.Context,
	operator authn.Principal,
	accountID string,
	profile OptionalDoctorProfileInput,
	expectedVersion int64,
	operationID, requestID string,
) (Account, []string, error) {
	profile, err := normalizedOptionalDoctorProfile(profile)
	if err != nil {
		return Account{}, nil, err
	}
	if profile.DisplayName == nil && profile.StaffNo == nil && profile.AvatarURL == nil && profile.Description == nil {
		return Account{}, nil, fmt.Errorf("%w: at least one doctor profile field is required", ErrInvalid)
	}
	return m.mutate(ctx, operator, accountID, expectedVersion, operationID, requestID,
		ActionUpdateDoctor, contractauthz.PermissionIdentityAuthorizationManage,
		func(tx TxStore, before Account) error {
			if before.AccountStatus != authn.AccountStatusActive || before.IdentityType() != IdentityTypeDoctor {
				return fmt.Errorf("%w: target is not an active doctor", ErrInvalidState)
			}
			return tx.UpdateDoctor(ctx, before.ID, profile, expectedVersion)
		})
}

func (m *Manager) ChangeDoctorDepartment(
	ctx context.Context,
	operator authn.Principal,
	accountID, departmentID string,
	expectedVersion int64,
	operationID, requestID string,
) (Account, []string, error) {
	departmentID, err := normalizedUUID(departmentID, "department_id")
	if err != nil {
		return Account{}, nil, err
	}
	return m.mutate(ctx, operator, accountID, expectedVersion, operationID, requestID,
		ActionChangeDoctorDepartment, contractauthz.PermissionIdentityAuthorizationManage,
		func(tx TxStore, before Account) error {
			if before.AccountStatus != authn.AccountStatusActive || before.IdentityType() != IdentityTypeDoctor {
				return fmt.Errorf("%w: target is not an active doctor", ErrInvalidState)
			}
			if before.DepartmentID == departmentID {
				return fmt.Errorf("%w: doctor already belongs to the department", ErrInvalidState)
			}
			return tx.ChangeDoctorDepartment(ctx, before.ID, departmentID, expectedVersion)
		})
}

func (m *Manager) RevokeDoctor(ctx context.Context, operator authn.Principal, accountID string, expectedVersion int64, operationID, requestID string) (Account, []string, error) {
	return m.mutate(ctx, operator, accountID, expectedVersion, operationID, requestID,
		ActionRevokeDoctor, contractauthz.PermissionIdentityAuthorizationManage,
		func(tx TxStore, before Account) error {
			if before.AccountStatus != authn.AccountStatusActive || before.IdentityType() != IdentityTypeDoctor {
				return fmt.Errorf("%w: target is not an active doctor", ErrInvalidState)
			}
			return tx.RevokeDoctor(ctx, before.ID, expectedVersion)
		})
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

func (m *Manager) mutate(
	ctx context.Context,
	operator authn.Principal,
	accountID string,
	expectedVersion int64,
	operationID, requestID, action, permission string,
	mutation func(TxStore, Account) error,
) (Account, []string, error) {
	if !operator.HasPermission(permission) {
		return Account{}, nil, ErrForbidden
	}
	accountID, err := normalizedUUID(accountID, "account_id")
	if err != nil {
		return Account{}, nil, err
	}
	operationID, err = normalizedUUID(operationID, "operation_id")
	if err != nil {
		return Account{}, nil, err
	}
	requestID = strings.TrimSpace(requestID)
	if len(requestID) > maxRequestIDBytes {
		return Account{}, nil, fmt.Errorf("%w: request_id exceeds %d bytes", ErrInvalid, maxRequestIDBytes)
	}
	if expectedVersion <= 0 {
		return Account{}, nil, fmt.Errorf("%w: management_version must be positive", ErrInvalid)
	}
	if operator.AccountID == accountID {
		return Account{}, nil, fmt.Errorf("%w: self management changes are not allowed", ErrForbidden)
	}

	var result Account
	var beforeVersion int64
	var replayed bool
	err = m.store.WithinIdentityAdminTransaction(ctx, func(tx TxStore) error {
		operation, exists, err := tx.FindIdentityAdminOperation(ctx, operationID)
		if err != nil {
			return err
		}
		if exists {
			if operation.OperatorAccountID != operator.AccountID || operation.TargetAccountID != accountID || operation.Action != action {
				return fmt.Errorf("%w: operation_id was already used for a different change", ErrConflict)
			}
			result, err = tx.GetIdentityAdminAccountForUpdate(ctx, accountID)
			replayed = true
			return err
		}

		before, err := tx.GetIdentityAdminAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if before.IdentityType() == IdentityTypeSuperAdmin {
			return fmt.Errorf("%w: super administrator accounts are read-only", ErrForbidden)
		}
		if before.ManagementVersion != expectedVersion {
			return ErrVersionConflict
		}
		beforeVersion = before.AuthorizationVersion
		if err := mutation(tx, before); err != nil {
			return err
		}
		after, err := tx.GetIdentityAdminAccountForUpdate(ctx, accountID)
		if err != nil {
			return err
		}
		if err := tx.RecordIdentityAdminChange(ctx, Change{
			OperationID: operationID, OperatorAccountID: operator.AccountID,
			TargetAccountID: accountID, Action: action, Before: before, After: after,
			RequestID: requestID,
		}); err != nil {
			return err
		}
		result = after
		return nil
	})
	if err != nil {
		return Account{}, nil, err
	}
	if (!replayed && result.AuthorizationVersion != beforeVersion) ||
		(replayed && actionChangesAuthorization(action)) {
		if err := m.versions.SetAuthorizationVersion(ctx, result.ID, result.AuthorizationVersion); err != nil {
			return Account{}, nil, fmt.Errorf("publish authorization version: %w", err)
		}
	}
	return result, availableActions(operator, result), nil
}

func actionChangesAuthorization(action string) bool {
	switch action {
	case ActionPromoteDoctor, ActionChangeDoctorDepartment, ActionRevokeDoctor,
		ActionDisableAccount, ActionEnableAccount:
		return true
	default:
		return false
	}
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

func normalizedPage(page, pageSize int64) (int64, int64, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 || pageSize < 1 || pageSize > maxPageSize {
		return 0, 0, fmt.Errorf("%w: page must be positive and page_size must be between 1 and %d", ErrInvalid, maxPageSize)
	}
	return page, pageSize, nil
}

func normalizedDoctorProfile(profile DoctorProfileInput) (DoctorProfileInput, error) {
	var err error
	profile.DisplayName, err = normalizedText(profile.DisplayName, "display_name", maxDisplayNameRunes, false)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	profile.StaffNo, err = normalizedText(profile.StaffNo, "staff_no", maxStaffNoRunes, true)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	profile.Description, err = normalizedText(profile.Description, "description", maxDescriptionRunes, true)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	profile.AvatarURL, err = normalizedAvatarURL(profile.AvatarURL)
	if err != nil {
		return DoctorProfileInput{}, err
	}
	return profile, nil
}

func normalizedOptionalDoctorProfile(profile OptionalDoctorProfileInput) (OptionalDoctorProfileInput, error) {
	var err error
	if profile.DisplayName != nil {
		value, valueErr := normalizedText(*profile.DisplayName, "display_name", maxDisplayNameRunes, false)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.DisplayName = &value
	}
	if profile.StaffNo != nil {
		value, valueErr := normalizedText(*profile.StaffNo, "staff_no", maxStaffNoRunes, true)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.StaffNo = &value
	}
	if profile.Description != nil {
		value, valueErr := normalizedText(*profile.Description, "description", maxDescriptionRunes, true)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.Description = &value
	}
	if profile.AvatarURL != nil {
		value, valueErr := normalizedAvatarURL(*profile.AvatarURL)
		if valueErr != nil {
			return OptionalDoctorProfileInput{}, valueErr
		}
		profile.AvatarURL = &value
	}
	return profile, err
}

func normalizedAvatarURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if len(value) > maxAvatarURLBytes {
		return "", fmt.Errorf("%w: avatar_url exceeds %d bytes", ErrInvalid, maxAvatarURLBytes)
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", fmt.Errorf("%w: avatar_url must be an HTTP(S) URL", ErrInvalid)
	}
	return value, nil
}

func normalizedText(value, field string, maxRunes int, allowEmpty bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" && !allowEmpty {
		return "", fmt.Errorf("%w: %s is required", ErrInvalid, field)
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return "", fmt.Errorf("%w: %s exceeds %d characters", ErrInvalid, field, maxRunes)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("%w: %s contains control characters", ErrInvalid, field)
	}
	return value, nil
}

func normalizedUUID(value, field string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("%w: %s must be a UUID", ErrInvalid, field)
	}
	return parsed.String(), nil
}

func validUUID(value, field string) error {
	_, err := normalizedUUID(value, field)
	return err
}

func normalizePhone(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(" ", "", "-", "").Replace(value)
	value = strings.TrimPrefix(value, "+86")
	if !mainlandPhone.MatchString(value) {
		return "", fmt.Errorf("%w: invalid phone number", ErrInvalid)
	}
	return "+86" + value, nil
}
