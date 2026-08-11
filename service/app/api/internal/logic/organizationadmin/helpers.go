package organizationadmin

import (
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/types"
)

func adminOrganizationUnitResponse(value *identityv1.AdminOrganizationUnit) *types.AdminOrganizationUnitResponse {
	return &types.AdminOrganizationUnitResponse{
		UnitID:      value.GetUnitId(),
		ParentID:    value.GetParentId(),
		UnitType:    value.GetUnitType(),
		Code:        value.GetCode(),
		Name:        value.GetName(),
		Status:      value.GetStatus(),
		ChildCount:  value.GetChildCount(),
		DoctorCount: value.GetDoctorCount(),
		Version:     value.GetVersion(),
	}
}

func adminOrganizationUnitsResponse(values []*identityv1.AdminOrganizationUnit) *types.ListOrganizationUnitsResponse {
	items := make([]types.AdminOrganizationUnitResponse, 0, len(values))
	for _, value := range values {
		items = append(items, *adminOrganizationUnitResponse(value))
	}
	return &types.ListOrganizationUnitsResponse{Items: items}
}
