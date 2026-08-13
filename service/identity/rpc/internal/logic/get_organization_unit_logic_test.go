package logic

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/organization"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/svc"
)

const (
	getOrganizationUnitOperatorID = "30000000-0000-0000-0000-000000000001"
	getOrganizationUnitID         = "30000000-0000-0000-0000-000000000002"
	getOrganizationParentID       = "30000000-0000-0000-0000-000000000003"
)

func TestGetOrganizationUnitLogic(t *testing.T) {
	store := &getOrganizationUnitStore{unit: organization.Unit{
		ID:          getOrganizationUnitID,
		ParentID:    getOrganizationParentID,
		Type:        organization.UnitTypeDepartment,
		Code:        "DEPT-CARDIOLOGY",
		Name:        "Cardiology",
		Status:      organization.StatusActive,
		ChildCount:  0,
		DoctorCount: 12,
		Version:     3,
	}}
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		AccountID: getOrganizationUnitOperatorID,
		Status:    authn.AccountStatusActive,
		Permissions: []string{
			contractauthz.PermissionIdentityDepartmentManage,
		},
	})
	logic := NewGetOrganizationUnitLogic(ctx, &svc.ServiceContext{
		Managers: svc.Managers{OrganizationUnit: organizationmanager.NewUnitManager(store)},
	})

	response, err := logic.GetOrganizationUnit(&identityv1.GetOrganizationUnitRequest{
		UnitId: getOrganizationUnitID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetUnitId() != getOrganizationUnitID ||
		response.GetParentId() != getOrganizationParentID ||
		response.GetUnitType() != string(organization.UnitTypeDepartment) ||
		response.GetCode() != "DEPT-CARDIOLOGY" ||
		response.GetName() != "Cardiology" ||
		response.GetStatus() != string(organization.StatusActive) ||
		response.GetDoctorCount() != 12 || response.GetVersion() != 3 {
		t.Fatalf("unexpected organization unit response: %#v", response)
	}
	if store.getUnitCalls != 1 {
		t.Fatalf("expected one store lookup, got %d", store.getUnitCalls)
	}
}

func TestGetOrganizationUnitLogicRequiresPermission(t *testing.T) {
	store := &getOrganizationUnitStore{}
	ctx := authn.ContextWithPrincipal(context.Background(), authn.Principal{
		AccountID: getOrganizationUnitOperatorID,
		Status:    authn.AccountStatusActive,
	})
	logic := NewGetOrganizationUnitLogic(ctx, &svc.ServiceContext{
		Managers: svc.Managers{OrganizationUnit: organizationmanager.NewUnitManager(store)},
	})

	_, err := logic.GetOrganizationUnit(&identityv1.GetOrganizationUnitRequest{
		UnitId: getOrganizationUnitID,
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
	if store.getUnitCalls != 0 {
		t.Fatalf("store must not be queried after permission denial")
	}
}

func TestGetOrganizationUnitLogicRequiresAuthentication(t *testing.T) {
	store := &getOrganizationUnitStore{}
	logic := NewGetOrganizationUnitLogic(context.Background(), &svc.ServiceContext{
		Managers: svc.Managers{OrganizationUnit: organizationmanager.NewUnitManager(store)},
	})

	_, err := logic.GetOrganizationUnit(&identityv1.GetOrganizationUnitRequest{
		UnitId: getOrganizationUnitID,
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
	if store.getUnitCalls != 0 {
		t.Fatalf("store must not be queried without authentication")
	}
}

type getOrganizationUnitStore struct {
	unit         organization.Unit
	getUnitCalls int
}

func (s *getOrganizationUnitStore) GetUnit(_ context.Context, unitID string) (organization.Unit, error) {
	s.getUnitCalls++
	if unitID != s.unit.ID {
		return organization.Unit{}, organization.ErrNotFound
	}
	return s.unit, nil
}

func (*getOrganizationUnitStore) ListUnits(context.Context, organization.ListFilter) ([]organization.Unit, error) {
	panic("ListUnits must not be called by GetOrganizationUnit")
}

func (*getOrganizationUnitStore) WithinOrganizationTransaction(context.Context, func(organization.TxStore) error) error {
	panic("WithinOrganizationTransaction must not be called by GetOrganizationUnit")
}
