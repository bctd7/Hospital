package identityadmin

import (
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/types"
)

func adminAccountSummaryResponse(value *identityv1.AdminAccountSummary) *types.AdminAccountSummaryResponse {
	if value == nil {
		return &types.AdminAccountSummaryResponse{}
	}
	return &types.AdminAccountSummaryResponse{
		AccountID: value.GetAccountId(), Nickname: value.Nickname,
		DisplayName: value.DisplayName, AvatarURL: value.AvatarUrl, MaskedPhone: value.MaskedPhone,
		AccountStatus: value.GetAccountStatus(), IdentityType: value.GetIdentityType(),
		DepartmentID: value.DepartmentId, DepartmentName: value.DepartmentName,
		ManagementVersion: value.GetManagementVersion(),
	}
}

func adminAccountDetailResponse(value *identityv1.AdminAccountDetail) *types.AdminAccountDetailResponse {
	summary := adminAccountSummaryResponse(value.GetSummary())
	return &types.AdminAccountDetailResponse{
		AccountID: summary.AccountID, Nickname: summary.Nickname, DisplayName: summary.DisplayName,
		AvatarURL: summary.AvatarURL, MaskedPhone: summary.MaskedPhone,
		AccountStatus: summary.AccountStatus, IdentityType: summary.IdentityType,
		DepartmentID: summary.DepartmentID, DepartmentName: summary.DepartmentName,
		ManagementVersion:       summary.ManagementVersion,
		PhoneVerificationStatus: value.PhoneVerificationStatus, StaffNo: value.StaffNo,
		Description: value.Description, Roles: append([]string{}, value.GetRoles()...),
		AuthorizationVersion: value.GetAuthorizationVersion(),
		AvailableActions:     append([]string{}, value.GetAvailableActions()...),
		CreatedAt:            value.GetCreatedAt(), UpdatedAt: value.GetUpdatedAt(),
		StaffStatus: value.StaffStatus,
	}
}

func adminAccountsPageResponse(value *identityv1.ListAdminAccountsResponse) *types.ListAdminAccountsResponse {
	items := make([]types.AdminAccountSummaryResponse, 0, len(value.GetItems()))
	for _, item := range value.GetItems() {
		items = append(items, *adminAccountSummaryResponse(item))
	}
	return &types.ListAdminAccountsResponse{
		Items: items, Page: value.GetPage(), PageSize: value.GetPageSize(), Total: value.GetTotal(),
	}
}

func managedAccountMutationRequest(value *types.ManagedAccountMutationRequest, requestID string) *identityv1.ManagedAccountMutationRequest {
	return &identityv1.ManagedAccountMutationRequest{
		AccountId: value.AccountID, ManagementVersion: value.ManagementVersion,
		OperationId: value.OperationID, RequestId: requestID,
	}
}
