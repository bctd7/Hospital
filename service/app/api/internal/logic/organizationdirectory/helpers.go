package organizationdirectory

import (
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/types"
)

func organizationContextResponse(value *identityv1.OrganizationContext) *types.OrganizationContextResponse {
	campuses := make([]types.CampusSummaryResponse, 0, len(value.GetCampuses()))
	for _, campus := range value.GetCampuses() {
		campuses = append(campuses, types.CampusSummaryResponse{
			CampusID:        campus.GetCampusId(),
			HospitalID:      campus.GetHospitalId(),
			Code:            campus.GetCode(),
			Name:            campus.GetName(),
			DepartmentCount: campus.GetDepartmentCount(),
			Status:          campus.GetStatus(),
			Version:         campus.GetVersion(),
		})
	}
	hospital := value.GetHospital()
	return &types.OrganizationContextResponse{
		Hospital: types.HospitalSummaryResponse{
			HospitalID: hospital.GetHospitalId(),
			Code:       hospital.GetCode(),
			Name:       hospital.GetName(),
			Version:    hospital.GetVersion(),
		},
		Campuses: campuses,
	}
}

func departmentsResponse(values []*identityv1.DepartmentSummary) *types.ListDepartmentsResponse {
	items := make([]types.DepartmentSummaryResponse, 0, len(values))
	for _, value := range values {
		items = append(items, types.DepartmentSummaryResponse{
			DepartmentID: value.GetDepartmentId(),
			CampusID:     value.GetCampusId(),
			CampusName:   value.GetCampusName(),
			Code:         value.GetCode(),
			Name:         value.GetName(),
			DoctorCount:  value.GetDoctorCount(),
			Status:       value.GetStatus(),
			Version:      value.GetVersion(),
		})
	}
	return &types.ListDepartmentsResponse{Items: items}
}
