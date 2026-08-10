package organizationadmin

import (
	"testing"

	identityv1 "hospital/contracts/gen/identity/v1"
)

func TestAdminOrganizationUnitResponse(t *testing.T) {
	response := adminOrganizationUnitResponse(&identityv1.AdminOrganizationUnit{
		UnitId: "department-1", ParentId: "campus-1", UnitType: "department",
		Code: "DEPT-A", Name: "Cardiology", Status: "active", DoctorCount: 3, Version: 2,
	})
	if response.UnitID != "department-1" || response.ParentID != "campus-1" ||
		response.DoctorCount != 3 || response.Version != 2 {
		t.Fatalf("unexpected response: %#v", response)
	}
}
