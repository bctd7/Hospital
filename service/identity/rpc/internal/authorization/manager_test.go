package authorization

import (
	"context"
	"errors"
	"sync"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

const (
	adminID      = "00000000-0000-0000-0000-000000000001"
	doctorID     = "00000000-0000-0000-0000-000000000002"
	departmentA  = "00000000-0000-0000-0000-000000000010"
	departmentB  = "00000000-0000-0000-0000-000000000011"
	operationOne = "00000000-0000-0000-0000-000000000020"
)

func TestAssignRoleAndReplay(t *testing.T) {
	store := newFakeStore()
	versions := &fakeVersionWriter{}
	manager, err := NewManager(store, versions)
	if err != nil {
		t.Fatal(err)
	}

	result, err := manager.AssignRole(context.Background(), store.principals[adminID], doctorID, authn.RoleSuperAdmin, operationOne, "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasRole(authn.RoleSuperAdmin) || result.AuthorizationVersion != 2 {
		t.Fatalf("unexpected result: %#v", result)
	}

	replayed, err := manager.AssignRole(context.Background(), store.principals[adminID], doctorID, authn.RoleSuperAdmin, operationOne, "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if replayed.AuthorizationVersion != 2 || len(store.changes) != 1 {
		t.Fatalf("replay created another change: version=%d changes=%d", replayed.AuthorizationVersion, len(store.changes))
	}
	if versions.calls != 2 || versions.accountID != doctorID || versions.version != 2 {
		t.Fatalf("authorization version projection was not repaired on replay: %#v", versions)
	}
}

func TestAuthorizationProjectionCanBeRepairedByIdempotentReplay(t *testing.T) {
	store := newFakeStore()
	versions := &fakeVersionWriter{err: errors.New("redis unavailable")}
	manager, err := NewManager(store, versions)
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.AssignRole(
		context.Background(), store.principals[adminID], doctorID,
		authn.RoleSuperAdmin, operationOne, "request-1",
	)
	if err == nil {
		t.Fatal("expected projection write failure")
	}
	if store.principals[doctorID].AuthorizationVersion != 2 || len(store.changes) != 1 {
		t.Fatal("database change should already be committed before projection update")
	}

	versions.err = nil
	result, err := manager.AssignRole(
		context.Background(), store.principals[adminID], doctorID,
		authn.RoleSuperAdmin, operationOne, "request-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.AuthorizationVersion != 2 || versions.version != 2 || len(store.changes) != 1 {
		t.Fatalf("idempotent replay did not repair projection: result=%#v writer=%#v", result, versions)
	}
}

func TestAuthorizationChangeRequiresPermission(t *testing.T) {
	store := newFakeStore()
	manager, err := NewManager(store, &fakeVersionWriter{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.ChangeStaffDepartment(context.Background(), store.principals[doctorID], adminID, departmentB,
		"00000000-0000-0000-0000-000000000021", "request-2")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestAuthorizationChangeRejectsSelfMutation(t *testing.T) {
	store := newFakeStore()
	manager, err := NewManager(store, &fakeVersionWriter{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.ChangeAccountStatus(context.Background(), store.principals[adminID], adminID, authn.AccountStatusDisabled,
		"00000000-0000-0000-0000-000000000022", "request-3")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestPromoteToDepartmentDoctorRequiresOfflineVerification(t *testing.T) {
	store := newFakeStore()
	manager, err := NewManager(store, &fakeVersionWriter{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.PromoteToDepartmentDoctor(
		context.Background(), store.principals[adminID], doctorID, departmentB, false, operationOne, "request-1",
	)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid offline verification, got %v", err)
	}
}

func TestPromoteToDepartmentDoctorSetsStaffRoleAndDepartment(t *testing.T) {
	store := newFakeStore()
	patient := store.principals[doctorID]
	patient.AccountType = authn.AccountTypePatient
	patient.Roles = nil
	patient.DepartmentID = ""
	store.principals[doctorID] = patient
	manager, err := NewManager(store, &fakeVersionWriter{})
	if err != nil {
		t.Fatal(err)
	}
	result, err := manager.PromoteToDepartmentDoctor(
		context.Background(), store.principals[adminID], doctorID, departmentB, true, operationOne, "request-1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.AccountType != authn.AccountTypeStaff || !result.HasRole(authn.RoleDepartmentDoctor) || result.DepartmentID != departmentB {
		t.Fatalf("unexpected promoted identity: %#v", result)
	}
}

type fakeStore struct {
	mu         sync.Mutex
	principals map[string]authn.Principal
	operations map[string]Operation
	changes    []Change
}

type fakeVersionWriter struct {
	accountID string
	version   int64
	calls     int
	err       error
}

func (w *fakeVersionWriter) SetAuthorizationVersion(_ context.Context, accountID string, version int64) error {
	w.accountID = accountID
	w.version = version
	w.calls++
	return w.err
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		principals: map[string]authn.Principal{
			adminID: {
				AccountID: adminID, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
				Roles: []string{authn.RoleSuperAdmin}, DepartmentID: departmentA,
				Permissions: []string{contractauthz.PermissionIdentityAuthorizationManage}, AuthorizationVersion: 1,
			},
			doctorID: {
				AccountID: doctorID, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
				Roles: []string{authn.RoleDepartmentDoctor}, DepartmentID: departmentA,
				Permissions: []string{contractauthz.PermissionAppointmentRead}, AuthorizationVersion: 1,
			},
		},
		operations: make(map[string]Operation),
	}
}

func (s *fakeStore) GetAuthorizationContext(_ context.Context, accountID string) (authn.Principal, error) {
	principal, ok := s.principals[accountID]
	if !ok {
		return authn.Principal{}, ErrNotFound
	}
	return principal, nil
}

func (s *fakeStore) WithinTransaction(_ context.Context, fn func(TxStore) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s)
}

func (s *fakeStore) FindOperation(_ context.Context, operationID string) (Operation, bool, error) {
	operation, exists := s.operations[operationID]
	return operation, exists, nil
}

func (s *fakeStore) SetRole(_ context.Context, accountID, roleCode string) error {
	principal, ok := s.principals[accountID]
	if !ok {
		return ErrNotFound
	}
	principal.Roles = []string{roleCode}
	principal.AuthorizationVersion++
	s.principals[accountID] = principal
	return nil
}

func (s *fakeStore) SetDepartment(_ context.Context, accountID, departmentID string) error {
	principal, ok := s.principals[accountID]
	if !ok {
		return ErrNotFound
	}
	principal.DepartmentID = departmentID
	principal.AuthorizationVersion++
	s.principals[accountID] = principal
	return nil
}

func (s *fakeStore) SetAccountStatus(_ context.Context, accountID, status string) error {
	principal, ok := s.principals[accountID]
	if !ok {
		return ErrNotFound
	}
	principal.Status = status
	principal.AuthorizationVersion++
	s.principals[accountID] = principal
	return nil
}

func (s *fakeStore) PromoteToDepartmentDoctor(_ context.Context, accountID, departmentID string, _ bool) error {
	principal := s.principals[accountID]
	principal.AccountType = authn.AccountTypeStaff
	principal.DepartmentID = departmentID
	principal.Roles = []string{authn.RoleDepartmentDoctor}
	principal.AuthorizationVersion++
	s.principals[accountID] = principal
	return nil
}

func (s *fakeStore) RecordChange(_ context.Context, change Change) error {
	s.operations[change.OperationID] = Operation{OperatorAccountID: change.OperatorAccountID, TargetAccountID: change.TargetAccountID}
	s.changes = append(s.changes, change)
	return nil
}
