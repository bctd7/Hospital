package organizationdirectory

import (
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/types"
)

func doctorsResponse(value *identityv1.ListDoctorsByDepartmentResponse) *types.ListDoctorsResponse {
	items := make([]types.DoctorSummaryResponse, 0, len(value.GetItems()))
	for _, item := range value.GetItems() {
		items = append(items, types.DoctorSummaryResponse{
			AccountID: item.GetAccountId(), DisplayName: item.GetDisplayName(),
			DepartmentID: item.GetDepartmentId(), AvatarURL: item.AvatarUrl,
			Description: item.Description, Version: item.GetVersion(),
		})
	}
	return &types.ListDoctorsResponse{
		Items: items, Page: value.GetPage(), PageSize: value.GetPageSize(), Total: value.GetTotal(),
	}
}
