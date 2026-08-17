package precedence_test

import (
	"context"
	"errors"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

const (
	itemA = "10000000-0000-4000-8000-000000000001"
	itemB = "10000000-0000-4000-8000-000000000002"
	itemC = "10000000-0000-4000-8000-000000000003"
	deptA = "20000000-0000-4000-8000-000000000001"
	deptB = "20000000-0000-4000-8000-000000000002"
	actor = "30000000-0000-4000-8000-000000000001"
)

func TestCreateRejectsIndirectCycle(t *testing.T) {
	store := &fakeStore{rules: []precedence.Rule{
		rule(itemA, itemB, deptA, deptA),
		rule(itemB, itemC, deptA, deptB),
	}}
	manager := newTestManager(t, store)
	_, err := manager.Create(context.Background(), superAdmin(), precedence.CreateInput{
		PredecessorItemID: itemC,
		SuccessorItemID:   itemA,
		StaffReason:       "构成循环的测试关系",
		OperationID:       "40000000-0000-4000-8000-000000000001",
	})
	if !errors.Is(err, precedence.ErrCycle) {
		t.Fatalf("expected ErrCycle, got %v", err)
	}
	if len(store.rules) != 2 {
		t.Fatalf("cycle rejection must not insert a rule, got %d", len(store.rules))
	}
}

func TestListDerivesTransitiveRelationWithoutPersistingIt(t *testing.T) {
	store := &fakeStore{rules: []precedence.Rule{
		rule(itemA, itemB, deptA, deptA),
		rule(itemB, itemC, deptA, deptB),
	}}
	manager := newTestManager(t, store)
	values, err := manager.List(context.Background(), superAdmin(), precedence.ListInput{
		ItemID: itemC, Direction: precedence.DirectionPredecessors, IncludeInferred: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 {
		t.Fatalf("expected direct and inferred predecessors, got %d", len(values))
	}
	foundDirect, foundInferred := false, false
	for _, value := range values {
		switch value.PredecessorItemID {
		case itemB:
			foundDirect = value.Direct && value.PathLength == 1
		case itemA:
			foundInferred = !value.Direct && value.PathLength == 2 && value.RuleID == ""
		}
	}
	if !foundDirect || !foundInferred {
		t.Fatalf("unexpected relations: %#v", values)
	}
	if len(store.rules) != 2 {
		t.Fatalf("inferred relation must not be persisted, got %d rules", len(store.rules))
	}
}

func TestDepartmentDoctorMayOnlyMaintainOwnedSuccessor(t *testing.T) {
	store := &fakeStore{}
	manager := newTestManager(t, store)
	doctor := authn.Principal{
		AccountID: actor, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
		Roles: []string{authn.RoleDepartmentDoctor}, DepartmentID: deptA,
		Permissions: []string{contractauthz.PermissionRuleEdit},
	}
	_, err := manager.Create(context.Background(), doctor, precedence.CreateInput{
		PredecessorItemID: itemA, SuccessorItemID: itemC,
		StaffReason: "后置项目属于其他科室", OperationID: "40000000-0000-4000-8000-000000000002",
	})
	if !errors.Is(err, precedence.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestCreateAllowsMultipleDirectPredecessors(t *testing.T) {
	store := &fakeStore{rules: []precedence.Rule{rule(itemA, itemC, deptA, deptB)}}
	manager := newTestManager(t, store)
	created, err := manager.Create(context.Background(), superAdmin(), precedence.CreateInput{
		PredecessorItemID: itemB, SuccessorItemID: itemC,
		StaffReason: "第二个独立前置项目", OperationID: "40000000-0000-4000-8000-000000000003",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.PredecessorItemID != itemB || created.SuccessorItemID != itemC || len(store.rules) != 2 {
		t.Fatalf("unexpected created rule: %#v, count=%d", created, len(store.rules))
	}
}

func TestDeleteDirectRuleRemovesDerivedRelation(t *testing.T) {
	first := rule(itemA, itemB, deptA, deptA)
	second := rule(itemB, itemC, deptA, deptB)
	store := &fakeStore{rules: []precedence.Rule{first, second}}
	manager := newTestManager(t, store)

	if err := manager.Delete(context.Background(), superAdmin(), precedence.DeleteInput{
		RuleID: first.RuleID, ExpectedVersion: 1,
		OperationID: "40000000-0000-4000-8000-000000000004",
	}); err != nil {
		t.Fatal(err)
	}
	values, err := manager.List(context.Background(), superAdmin(), precedence.ListInput{
		ItemID: itemC, Direction: precedence.DirectionPredecessors, IncludeInferred: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].PredecessorItemID != itemB || !values[0].Direct {
		t.Fatalf("deleted edge must remove its derived relation, got %#v", values)
	}
}

func TestUpdateRequiresCurrentVersion(t *testing.T) {
	existing := rule(itemA, itemB, deptA, deptA)
	store := &fakeStore{rules: []precedence.Rule{existing}}
	manager := newTestManager(t, store)

	updated, err := manager.Update(context.Background(), superAdmin(), precedence.UpdateInput{
		RuleID: existing.RuleID, StaffReason: "更新后的专业原因", PatientMessage: "请先完成项目A",
		ExpectedVersion: 1, OperationID: "40000000-0000-4000-8000-000000000005",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != 2 || updated.StaffReason != "更新后的专业原因" {
		t.Fatalf("unexpected updated rule: %#v", updated)
	}
	_, err = manager.Update(context.Background(), superAdmin(), precedence.UpdateInput{
		RuleID: existing.RuleID, StaffReason: "使用旧版本再次修改",
		ExpectedVersion: 1, OperationID: "40000000-0000-4000-8000-000000000006",
	})
	if !errors.Is(err, precedence.ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}
}

func newTestManager(t *testing.T, store *fakeStore) *precedence.Manager {
	t.Helper()
	directory := fakeDirectory{projects: map[string]precedence.ProjectReference{
		itemA: {ItemID: itemA, DepartmentID: deptA, Name: "项目A"},
		itemB: {ItemID: itemB, DepartmentID: deptA, Name: "项目B"},
		itemC: {ItemID: itemC, DepartmentID: deptB, Name: "项目C"},
	}}
	manager, err := precedence.New(store, directory)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}

func superAdmin() authn.Principal {
	return authn.Principal{
		AccountID: actor, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
		Roles: []string{authn.RoleSuperAdmin}, Permissions: []string{contractauthz.PermissionRuleRead, contractauthz.PermissionRuleEdit},
	}
}

func rule(from, to, fromDepartment, toDepartment string) precedence.Rule {
	return precedence.Rule{
		RuleID:            "50000000-0000-4000-8000-0000000000" + from[len(from)-1:] + to[len(to)-1:],
		PredecessorItemID: from, PredecessorDepartmentID: fromDepartment, PredecessorItemName: "项目" + from[len(from)-1:],
		SuccessorItemID: to, SuccessorDepartmentID: toDepartment, SuccessorItemName: "项目" + to[len(to)-1:],
		StaffReason: "测试规则", Version: 1,
	}
}

type fakeDirectory struct {
	projects map[string]precedence.ProjectReference
}

func (d fakeDirectory) ResolveProject(_ context.Context, itemID string) (precedence.ProjectReference, error) {
	value, ok := d.projects[itemID]
	if !ok {
		return precedence.ProjectReference{}, precedence.ErrNotFound
	}
	return value, nil
}

type fakeStore struct{ rules []precedence.Rule }

func (s *fakeStore) ListAll(context.Context) ([]precedence.Rule, error) {
	return append([]precedence.Rule(nil), s.rules...), nil
}

func (s *fakeStore) WithinWriteTransaction(ctx context.Context, fn func(precedence.TxStore) error) error {
	return fn((*fakeTx)(s))
}

type fakeTx fakeStore

func (s *fakeTx) LockGraph(context.Context) error { return nil }
func (s *fakeTx) ListAll(context.Context) ([]precedence.Rule, error) {
	return append([]precedence.Rule(nil), s.rules...), nil
}
func (s *fakeTx) GetByIDForUpdate(_ context.Context, ruleID string) (precedence.Rule, error) {
	for _, value := range s.rules {
		if value.RuleID == ruleID {
			return value, nil
		}
	}
	return precedence.Rule{}, precedence.ErrNotFound
}
func (s *fakeTx) GetByPair(_ context.Context, predecessor, successor string) (precedence.Rule, error) {
	for _, value := range s.rules {
		if value.PredecessorItemID == predecessor && value.SuccessorItemID == successor {
			return value, nil
		}
	}
	return precedence.Rule{}, precedence.ErrNotFound
}
func (s *fakeTx) GetByCreateOperationID(_ context.Context, operationID string) (precedence.Rule, error) {
	for _, value := range s.rules {
		if value.CreateOperationID == operationID && operationID != "" {
			return value, nil
		}
	}
	return precedence.Rule{}, precedence.ErrNotFound
}
func (s *fakeTx) Insert(_ context.Context, value precedence.Rule) (precedence.Rule, error) {
	value.Version = 1
	s.rules = append(s.rules, value)
	return value, nil
}
func (s *fakeTx) UpdateText(_ context.Context, ruleID, staffReason, patientMessage string, expectedVersion int64) (precedence.Rule, error) {
	for index, value := range s.rules {
		if value.RuleID != ruleID {
			continue
		}
		if value.Version != expectedVersion {
			return precedence.Rule{}, precedence.ErrVersionConflict
		}
		value.StaffReason = staffReason
		value.PatientMessage = patientMessage
		value.Version++
		s.rules[index] = value
		return value, nil
	}
	return precedence.Rule{}, precedence.ErrVersionConflict
}

func (s *fakeTx) Delete(_ context.Context, ruleID string, expectedVersion int64) error {
	for index, value := range s.rules {
		if value.RuleID != ruleID {
			continue
		}
		if value.Version != expectedVersion {
			return precedence.ErrVersionConflict
		}
		s.rules = append(s.rules[:index], s.rules[index+1:]...)
		return nil
	}
	return precedence.ErrVersionConflict
}

var _ precedence.Store = (*fakeStore)(nil)
var _ precedence.TxStore = (*fakeTx)(nil)
