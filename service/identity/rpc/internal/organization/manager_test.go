package organization

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

const (
	organizationAdminID = "00000000-0000-0000-0000-000000000001"
	organizationUserID  = "00000000-0000-0000-0000-000000000002"
	hospitalID          = "00000000-0000-0000-0000-000000000010"
	campusAID           = "00000000-0000-0000-0000-000000000011"
	campusBID           = "00000000-0000-0000-0000-000000000012"
	departmentAID       = "00000000-0000-0000-0000-000000000013"
	createOperationID   = "00000000-0000-0000-0000-000000000020"
	updateOperationID   = "00000000-0000-0000-0000-000000000021"
	disableOperationID  = "00000000-0000-0000-0000-000000000022"
	enableOperationID   = "00000000-0000-0000-0000-000000000023"
)

func TestCreateUnitAndReplay(t *testing.T) {
	store := newFakeOrganizationStore()
	manager := NewManager(store)
	command := CreateUnitCommand{
		Type: UnitTypeCampus, ParentID: hospitalID, Name: "东院区",
		OperationID: createOperationID, RequestID: "request-create",
	}

	created, err := manager.CreateUnit(context.Background(), organizationAdmin(), command)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.ParentID != hospitalID || created.Type != UnitTypeCampus ||
		created.Name != "东院区" || created.Status != StatusActive || created.Version != 1 {
		t.Fatalf("unexpected created unit: %#v", created)
	}
	if created.Code != "CAMPUS-000000000000" {
		t.Fatalf("unexpected generated code: %q", created.Code)
	}

	replayed, err := manager.CreateUnit(context.Background(), organizationAdmin(), command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ID != created.ID || len(store.changes) != 1 {
		t.Fatalf("replay was not idempotent: result=%#v changes=%d", replayed, len(store.changes))
	}
}

func TestCreateUnitRequiresPermission(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())
	_, err := manager.CreateUnit(context.Background(), authn.Principal{
		AccountID: organizationUserID,
		Status:    authn.AccountStatusActive,
	}, CreateUnitCommand{
		Type: UnitTypeCampus, ParentID: hospitalID, Name: "东院区",
		OperationID: createOperationID,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestGetManagedUnit(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())

	unit, err := manager.GetManagedUnit(context.Background(), organizationAdmin(), campusAID)
	if err != nil {
		t.Fatal(err)
	}
	if unit.ID != campusAID || unit.Type != UnitTypeCampus {
		t.Fatalf("unexpected managed unit: %#v", unit)
	}
}

func TestGetManagedUnitRequiresPermission(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())

	_, err := manager.GetManagedUnit(context.Background(), authn.Principal{
		AccountID: organizationUserID,
		Status:    authn.AccountStatusActive,
	}, campusAID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestGetManagedUnitRejectsHospitalRoot(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())

	_, err := manager.GetManagedUnit(context.Background(), organizationAdmin(), hospitalID)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestGetDirectoryContextReturnsOnlyActiveCampuses(t *testing.T) {
	store := newFakeOrganizationStore()
	disabled := store.units[campusBID]
	disabled.Status = StatusDisabled
	store.units[campusBID] = disabled
	manager := NewManager(store)

	value, err := manager.GetDirectoryContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if value.Hospital.ID != hospitalID {
		t.Fatalf("unexpected hospital: %#v", value.Hospital)
	}
	if len(value.Campuses) != 1 || value.Campuses[0].ID != campusAID {
		t.Fatalf("unexpected public campuses: %#v", value.Campuses)
	}
}

func TestListDirectoryDepartmentsIncludesAuthoritativeCampus(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())

	departments, err := manager.ListDirectoryDepartments(context.Background(), campusAID)
	if err != nil {
		t.Fatal(err)
	}
	if len(departments) != 1 || departments[0].ID != departmentAID ||
		departments[0].CampusName == "" {
		t.Fatalf("unexpected public departments: %#v", departments)
	}
}

func TestListManagedUnitsRequiresPermissionAndParent(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())
	parentID := hospitalID
	active := StatusActive

	_, err := manager.ListManagedUnits(context.Background(), authn.Principal{
		AccountID: organizationUserID,
		Status:    authn.AccountStatusActive,
	}, ListFilter{Type: UnitTypeCampus, ParentID: &parentID, Status: &active})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	_, err = manager.ListManagedUnits(context.Background(), organizationAdmin(), ListFilter{
		Type: UnitTypeDepartment,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestListManagedUnitsRejectsHospitalType(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())
	parentID := hospitalID

	_, err := manager.ListManagedUnits(context.Background(), organizationAdmin(), ListFilter{
		Type: UnitTypeHospital, ParentID: &parentID,
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestCreateDepartmentRejectsHospitalParent(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())
	_, err := manager.CreateUnit(context.Background(), organizationAdmin(), CreateUnitCommand{
		Type: UnitTypeDepartment, ParentID: hospitalID, Name: "放射科",
		OperationID: createOperationID,
	})
	if !errors.Is(err, ErrInvalidHierarchy) {
		t.Fatalf("expected ErrInvalidHierarchy, got %v", err)
	}
}

func TestUpdateDepartmentNameAndCampus(t *testing.T) {
	store := newFakeOrganizationStore()
	manager := NewManager(store)
	name := "医学影像科"
	parentID := campusBID

	updated, err := manager.UpdateUnit(context.Background(), organizationAdmin(), UpdateUnitCommand{
		UnitID: departmentAID, Name: &name, ParentID: &parentID,
		ExpectedVersion: 1, OperationID: updateOperationID, RequestID: "request-update",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.ParentID != campusBID || updated.Version != 2 {
		t.Fatalf("unexpected updated unit: %#v", updated)
	}
	if len(store.changes) != 1 || store.changes[0].Before == nil ||
		store.changes[0].Before.ParentID != campusAID {
		t.Fatalf("unexpected change record: %#v", store.changes)
	}
}

func TestUpdateUnitRejectsStaleVersion(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())
	name := "医学影像科"
	_, err := manager.UpdateUnit(context.Background(), organizationAdmin(), UpdateUnitCommand{
		UnitID: departmentAID, Name: &name,
		ExpectedVersion: 99, OperationID: updateOperationID,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func TestDisableCampusRejectsActiveDepartment(t *testing.T) {
	manager := NewManager(newFakeOrganizationStore())
	_, err := manager.DisableUnit(context.Background(), organizationAdmin(), ChangeUnitStatusCommand{
		UnitID: campusAID, ExpectedVersion: 1, OperationID: disableOperationID,
	})
	if !errors.Is(err, ErrActiveChildren) {
		t.Fatalf("expected ErrActiveChildren, got %v", err)
	}
}

func TestDisableDepartmentRejectsActiveDoctor(t *testing.T) {
	store := newFakeOrganizationStore()
	store.activeDoctors[departmentAID] = 1
	manager := NewManager(store)

	_, err := manager.DisableUnit(context.Background(), organizationAdmin(), ChangeUnitStatusCommand{
		UnitID: departmentAID, ExpectedVersion: 1, OperationID: disableOperationID,
	})
	if !errors.Is(err, ErrActiveDoctors) {
		t.Fatalf("expected ErrActiveDoctors, got %v", err)
	}
}

func TestEnableDepartmentRequiresActiveCampus(t *testing.T) {
	store := newFakeOrganizationStore()
	campus := store.units[campusAID]
	campus.Status = StatusDisabled
	store.units[campusAID] = campus
	department := store.units[departmentAID]
	department.Status = StatusDisabled
	store.units[departmentAID] = department
	manager := NewManager(store)

	_, err := manager.EnableUnit(context.Background(), organizationAdmin(), ChangeUnitStatusCommand{
		UnitID: departmentAID, ExpectedVersion: 1, OperationID: enableOperationID,
	})
	if !errors.Is(err, ErrInvalidHierarchy) {
		t.Fatalf("expected ErrInvalidHierarchy, got %v", err)
	}
}

func organizationAdmin() authn.Principal {
	return authn.Principal{
		AccountID: organizationAdminID,
		Status:    authn.AccountStatusActive,
		Roles:     []string{authn.RoleSuperAdmin},
		Permissions: []string{
			contractauthz.PermissionIdentityDepartmentManage,
		},
	}
}

type fakeOrganizationStore struct {
	mu            sync.Mutex
	units         map[string]Unit
	operations    map[string]Operation
	changes       []Change
	activeDoctors map[string]int64
}

func newFakeOrganizationStore() *fakeOrganizationStore {
	return &fakeOrganizationStore{
		units: map[string]Unit{
			hospitalID: {
				ID: hospitalID, Type: UnitTypeHospital, Code: "HOSPITAL",
				Name: "演示医院", Status: StatusActive, Version: 1,
			},
			campusAID: {
				ID: campusAID, ParentID: hospitalID, Type: UnitTypeCampus, Code: "CAMPUS-A",
				Name: "本部院区", Status: StatusActive, Version: 1,
			},
			campusBID: {
				ID: campusBID, ParentID: hospitalID, Type: UnitTypeCampus, Code: "CAMPUS-B",
				Name: "东院区", Status: StatusActive, Version: 1,
			},
			departmentAID: {
				ID: departmentAID, ParentID: campusAID, Type: UnitTypeDepartment, Code: "DEPT-A",
				Name: "放射科", Status: StatusActive, Version: 1,
			},
		},
		operations:    make(map[string]Operation),
		activeDoctors: make(map[string]int64),
	}
}

func (s *fakeOrganizationStore) GetUnit(_ context.Context, unitID string) (Unit, error) {
	unit, ok := s.units[unitID]
	if !ok {
		return Unit{}, ErrNotFound
	}
	return unit, nil
}

func (s *fakeOrganizationStore) ListUnits(_ context.Context, filter ListFilter) ([]Unit, error) {
	result := make([]Unit, 0)
	for _, unit := range s.units {
		if unit.Type != filter.Type {
			continue
		}
		if filter.ParentID != nil && unit.ParentID != *filter.ParentID {
			continue
		}
		if filter.Status != nil && unit.Status != *filter.Status {
			continue
		}
		result = append(result, unit)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (s *fakeOrganizationStore) WithinOrganizationTransaction(_ context.Context, fn func(TxStore) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s)
}

func (s *fakeOrganizationStore) FindOperation(_ context.Context, operationID string) (Operation, bool, error) {
	operation, exists := s.operations[operationID]
	return operation, exists, nil
}

func (s *fakeOrganizationStore) GetUnitForUpdate(ctx context.Context, unitID string) (Unit, error) {
	return s.GetUnit(ctx, unitID)
}

func (s *fakeOrganizationStore) CreateUnit(_ context.Context, unit Unit) error {
	if _, exists := s.units[unit.ID]; exists {
		return ErrConflict
	}
	for _, current := range s.units {
		if current.ParentID == unit.ParentID && current.Code == unit.Code {
			return ErrConflict
		}
	}
	s.units[unit.ID] = unit
	return nil
}

func (s *fakeOrganizationStore) UpdateUnit(_ context.Context, unit Unit, expectedVersion int64) error {
	current, exists := s.units[unit.ID]
	if !exists {
		return ErrNotFound
	}
	if current.Version != expectedVersion {
		return ErrVersionConflict
	}
	s.units[unit.ID] = unit
	return nil
}

func (s *fakeOrganizationStore) SetUnitStatus(_ context.Context, unitID string, status Status, expectedVersion int64) error {
	unit, exists := s.units[unitID]
	if !exists {
		return ErrNotFound
	}
	if unit.Version != expectedVersion {
		return ErrVersionConflict
	}
	unit.Status = status
	unit.Version++
	s.units[unitID] = unit
	return nil
}

func (s *fakeOrganizationStore) CountActiveChildren(_ context.Context, unitID string) (int64, error) {
	var count int64
	for _, unit := range s.units {
		if unit.ParentID == unitID && unit.Status == StatusActive {
			count++
		}
	}
	return count, nil
}

func (s *fakeOrganizationStore) CountActiveDoctors(_ context.Context, departmentID string) (int64, error) {
	return s.activeDoctors[departmentID], nil
}

func (s *fakeOrganizationStore) RecordChange(_ context.Context, change Change) error {
	s.operations[change.OperationID] = Operation{
		OperatorAccountID: change.OperatorAccountID,
		UnitID:            change.UnitID,
		Action:            change.Action,
		Result:            change.After,
	}
	s.changes = append(s.changes, change)
	return nil
}
