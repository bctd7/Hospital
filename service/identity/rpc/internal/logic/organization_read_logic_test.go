package logic

import (
	"context"
	"sort"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/organization"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/svc"
)

const (
	readLogicHospitalID   = "40000000-0000-0000-0000-000000000001"
	readLogicCampusID     = "40000000-0000-0000-0000-000000000002"
	readLogicCampusTwoID  = "40000000-0000-0000-0000-000000000003"
	readLogicDepartmentID = "40000000-0000-0000-0000-000000000004"
	readLogicAdminID      = "40000000-0000-0000-0000-000000000005"
)

func TestGetOrganizationContextLogicIsPublic(t *testing.T) {
	logic := NewGetOrganizationContextLogic(context.Background(), readOrganizationServiceContext())

	response, err := logic.GetOrganizationContext(&identityv1.GetOrganizationContextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetHospital().GetHospitalId() != readLogicHospitalID {
		t.Fatalf("unexpected hospital: %#v", response.GetHospital())
	}
	if len(response.GetCampuses()) != 1 || response.GetCampuses()[0].GetCampusId() != readLogicCampusID {
		t.Fatalf("public directory exposed unexpected campuses: %#v", response.GetCampuses())
	}
}

func TestListDepartmentsLogicReturnsCampusName(t *testing.T) {
	logic := NewListDepartmentsLogic(context.Background(), readOrganizationServiceContext())

	response, err := logic.ListDepartments(&identityv1.ListDepartmentsRequest{CampusId: readLogicCampusID})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetItems()) != 1 || response.GetItems()[0].GetDepartmentId() != readLogicDepartmentID ||
		response.GetItems()[0].GetCampusName() != "Main Campus" {
		t.Fatalf("unexpected departments: %#v", response.GetItems())
	}
}

func TestListOrganizationUnitsLogicSupportsAllStatuses(t *testing.T) {
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		AccountID: readLogicAdminID,
		Status:    authn.AccountStatusActive,
		Permissions: []string{
			contractauthz.PermissionIdentityDepartmentManage,
		},
	})
	logic := NewListOrganizationUnitsLogic(ctx, readOrganizationServiceContext())

	response, err := logic.ListOrganizationUnits(&identityv1.ListOrganizationUnitsRequest{
		UnitType: string(organization.UnitTypeCampus), ParentId: readLogicHospitalID, Status: "all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.GetItems()) != 2 {
		t.Fatalf("expected active and disabled campuses, got %#v", response.GetItems())
	}
}

func readOrganizationServiceContext() *svc.ServiceContext {
	store := &readOrganizationStore{units: map[string]organization.Unit{
		readLogicHospitalID: {
			ID: readLogicHospitalID, Type: organization.UnitTypeHospital,
			Code: "HOSPITAL", Name: "Hospital", Status: organization.StatusActive, Version: 1,
		},
		readLogicCampusID: {
			ID: readLogicCampusID, ParentID: readLogicHospitalID, Type: organization.UnitTypeCampus,
			Code: "CAMPUS-A", Name: "Main Campus", Status: organization.StatusActive, ChildCount: 1, Version: 1,
		},
		readLogicCampusTwoID: {
			ID: readLogicCampusTwoID, ParentID: readLogicHospitalID, Type: organization.UnitTypeCampus,
			Code: "CAMPUS-B", Name: "Disabled Campus", Status: organization.StatusDisabled, Version: 2,
		},
		readLogicDepartmentID: {
			ID: readLogicDepartmentID, ParentID: readLogicCampusID, Type: organization.UnitTypeDepartment,
			Code: "DEPT-A", Name: "Cardiology", Status: organization.StatusActive, DoctorCount: 2, Version: 1,
		},
	}}
	return &svc.ServiceContext{
		Managers: svc.Managers{
			OrganizationUnit:      organizationmanager.NewUnitManager(store),
			OrganizationDirectory: organizationmanager.NewDirectoryManager(store),
		},
	}
}

type readOrganizationStore struct {
	units map[string]organization.Unit
}

func (s *readOrganizationStore) GetUnit(_ context.Context, unitID string) (organization.Unit, error) {
	unit, ok := s.units[unitID]
	if !ok {
		return organization.Unit{}, organization.ErrNotFound
	}
	return unit, nil
}

func (s *readOrganizationStore) ListUnits(_ context.Context, filter organization.ListFilter) ([]organization.Unit, error) {
	units := make([]organization.Unit, 0)
	for _, unit := range s.units {
		if unit.Type != filter.Type || filter.ParentID != nil && unit.ParentID != *filter.ParentID ||
			filter.Status != nil && unit.Status != *filter.Status {
			continue
		}
		units = append(units, unit)
	}
	sort.Slice(units, func(i, j int) bool {
		if units[i].Code == units[j].Code {
			return units[i].ID < units[j].ID
		}
		return units[i].Code < units[j].Code
	})
	return units, nil
}

func (*readOrganizationStore) WithinOrganizationTransaction(context.Context, func(organization.TxStore) error) error {
	panic("organization read logic must not start a transaction")
}
