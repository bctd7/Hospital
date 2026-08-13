package logic

import (
	"errors"
	"time"

	identityv1 "hospital/contracts/gen/identity/v1"
	accountmanager "hospital/service/identity/rpc/internal/account/manager"

	"google.golang.org/grpc/codes"
)

func accountManagementRPCError(err error) error {
	switch {
	case errors.Is(err, accountmanager.ErrInvalid):
		return organizationStatusError(codes.InvalidArgument, err.Error(), "INVALID_IDENTITY_MANAGEMENT_REQUEST")
	case errors.Is(err, accountmanager.ErrForbidden):
		return organizationStatusError(codes.PermissionDenied, "permission denied", "PERMISSION_DENIED")
	case errors.Is(err, accountmanager.ErrNotFound):
		return organizationStatusError(codes.NotFound, "identity resource not found", "ACCOUNT_NOT_FOUND")
	case errors.Is(err, accountmanager.ErrConflict):
		return organizationStatusError(codes.AlreadyExists, err.Error(), "OPERATION_ID_REUSED")
	case errors.Is(err, accountmanager.ErrVersionConflict):
		return organizationStatusError(codes.Aborted, "identity management version conflict", "VERSION_CONFLICT")
	case errors.Is(err, accountmanager.ErrInvalidState):
		return organizationStatusError(codes.FailedPrecondition, err.Error(), "INVALID_ACCOUNT_STATE")
	default:
		return organizationStatusError(codes.Internal, "identity management operation failed", "IDENTITY_MANAGEMENT_FAILED")
	}
}

func identityDirectoryRPCError(err error) error {
	if errors.Is(err, accountmanager.ErrNotFound) {
		return organizationStatusError(codes.NotFound, "department not found", "DEPARTMENT_NOT_FOUND")
	}
	return accountManagementRPCError(err)
}

func accountDisplayProfileResponse(value accountmanager.DisplayProfile) *identityv1.AccountDisplayProfile {
	return &identityv1.AccountDisplayProfile{
		Nickname: optionalProtoString(value.Nickname), ManagementVersion: value.ManagementVersion,
	}
}

func doctorSummaryResponse(value accountmanager.DoctorSummary) *identityv1.DoctorSummary {
	return &identityv1.DoctorSummary{
		AccountId: value.AccountID, DisplayName: value.DisplayName, DepartmentId: value.DepartmentID,
		AvatarUrl: optionalProtoString(value.AvatarURL), Description: optionalProtoString(value.Description),
		Version: value.ManagementVersion,
	}
}

func doctorsPageResponse(value accountmanager.DoctorPage) *identityv1.ListDoctorsByDepartmentResponse {
	items := make([]*identityv1.DoctorSummary, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, doctorSummaryResponse(item))
	}
	return &identityv1.ListDoctorsByDepartmentResponse{
		Items: items, Page: value.Page, PageSize: value.PageSize, Total: value.Total,
	}
}

func adminAccountSummaryResponse(value accountmanager.Account) *identityv1.AdminAccountSummary {
	departmentID := value.DepartmentID
	departmentName := value.DepartmentName
	if value.StaffStatus != accountmanager.StaffStatusActive {
		departmentID = ""
		departmentName = ""
	}
	return &identityv1.AdminAccountSummary{
		AccountId: value.ID, Nickname: optionalProtoString(value.Nickname),
		DisplayName: optionalProtoString(value.DisplayName), AvatarUrl: optionalProtoString(value.AvatarURL),
		MaskedPhone: optionalProtoString(value.MaskedPhone), AccountStatus: value.AccountStatus,
		IdentityType: value.IdentityType(), DepartmentId: optionalProtoString(departmentID),
		DepartmentName: optionalProtoString(departmentName), ManagementVersion: value.ManagementVersion,
	}
}

func adminAccountDetailResponse(value accountmanager.Account, actions []string) *identityv1.AdminAccountDetail {
	return &identityv1.AdminAccountDetail{
		Summary:                 adminAccountSummaryResponse(value),
		PhoneVerificationStatus: optionalProtoString(value.PhoneVerificationStatus),
		StaffNo:                 optionalProtoString(value.StaffNo), Description: optionalProtoString(value.Description),
		Roles: append([]string(nil), value.Roles...), AuthorizationVersion: value.AuthorizationVersion,
		AvailableActions: append([]string(nil), actions...),
		CreatedAt:        value.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        value.UpdatedAt.UTC().Format(time.RFC3339Nano),
		StaffStatus:      optionalProtoString(value.StaffStatus),
	}
}

func adminAccountsPageResponse(value accountmanager.AccountPage) *identityv1.ListAdminAccountsResponse {
	items := make([]*identityv1.AdminAccountSummary, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, adminAccountSummaryResponse(item))
	}
	return &identityv1.ListAdminAccountsResponse{
		Items: items, Page: value.Page, PageSize: value.PageSize, Total: value.Total,
	}
}

func optionalProtoString(value string) *string {
	if value == "" {
		return nil
	}
	result := value
	return &result
}
