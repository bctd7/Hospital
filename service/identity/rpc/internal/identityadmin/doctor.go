package identityadmin

import (
	"context"
	"fmt"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

// ListDoctors is the public read model for active doctors in a department.
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
