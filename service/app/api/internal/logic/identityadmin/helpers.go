package identityadmin

import (
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/types"
)

func identityResponse(value *identityv1.AuthorizationContext) types.CurrentIdentityResponse {
	return types.CurrentIdentityResponse{
		AccountID: value.GetAccountId(), AccountType: value.GetAccountType(), Status: value.GetStatus(),
		Roles: value.GetRoles(), DepartmentID: value.GetDepartmentId(), Permissions: value.GetPermissions(),
		AuthorizationVersion: value.GetAuthorizationVersion(),
	}
}
