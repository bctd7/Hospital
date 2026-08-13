package manager

import (
	"context"
	"errors"
	"sort"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

const catalogOperatorID = "00000000-0000-0000-0000-000000000020"

type stubStore struct{}

func (stubStore) GetItem(context.Context, string) (ExaminationItem, error) {
	return ExaminationItem{}, ErrNotImplemented
}

func (stubStore) ListItems(context.Context, ProjectListFilter) ([]ExaminationItem, int64, error) {
	return nil, 0, ErrNotImplemented
}

func (stubStore) WithinProjectTransaction(context.Context, func(ProjectTxStore) error) error {
	return ErrNotImplemented
}

func TestStaffManagerRequiresProjectStore(t *testing.T) {
	if _, err := NewStaffManager(nil, nil, nil); err == nil {
		t.Fatal("expected nil project store to be rejected")
	}
	if manager, _ := projectTestManager(stubStore{}); manager == nil {
		t.Fatal("expected staff manager")
	}
}

func projectTestManager(store ProjectStore) (*StaffManager, error) {
	return &StaffManager{projectStore: store}, nil
}

func TestCreatePersistsActiveItemAndAudit(t *testing.T) {
	store := newMemoryCatalogStore()
	manager, err := projectTestManager(store)
	if err != nil {
		t.Fatal(err)
	}

	created, err := manager.CreateProject(context.Background(), catalogOperator(), CreateProjectCommand{
		OwnerDepartmentID: " " + validationDepartmentID + " ",
		Name:              " 腹部 CT ",
		Description:       " 检查前禁食。\n可少量饮水。 ",
		OperationID:       validationOperationID,
		RequestID:         " request-create ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ItemID == "" || created.OwnerDepartmentID != validationDepartmentID ||
		created.Name != "腹部 CT" || created.Description != "检查前禁食。\n可少量饮水。" ||
		created.Status != StatusActive || created.Version != 1 ||
		created.CreatedAt.IsZero() || !created.CreatedAt.Equal(created.UpdatedAt) {
		t.Fatalf("unexpected created item: %#v", created)
	}
	if store.createCalls != 1 || store.item.ItemID != created.ItemID {
		t.Fatalf("item was not created once: calls=%d item=%#v", store.createCalls, store.item)
	}
	if store.change == nil || store.change.Before != nil || store.change.After.ItemID != created.ItemID ||
		store.change.Action != ActionExaminationItemCreated || store.change.RequestID != "request-create" {
		t.Fatalf("unexpected recorded change: %#v", store.change)
	}
}

func TestCreateReplaysSameOperation(t *testing.T) {
	store := newMemoryCatalogStore()
	manager, _ := projectTestManager(store)
	command := validCreateCommand()

	created, err := manager.CreateProject(context.Background(), catalogOperator(), command)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := manager.CreateProject(context.Background(), catalogOperator(), command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ItemID != created.ItemID || store.createCalls != 1 {
		t.Fatalf("replay=%#v created=%#v createCalls=%d", replayed, created, store.createCalls)
	}
}

func TestCreateRejectsOperationReuseWithDifferentRequest(t *testing.T) {
	store := newMemoryCatalogStore()
	manager, _ := projectTestManager(store)
	command := validCreateCommand()
	if _, err := manager.CreateProject(context.Background(), catalogOperator(), command); err != nil {
		t.Fatal(err)
	}
	command.Description = "不同描述"
	if _, err := manager.CreateProject(context.Background(), catalogOperator(), command); !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if store.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", store.createCalls)
	}
}

func TestCreateRejectsCrossDepartmentOperator(t *testing.T) {
	store := newMemoryCatalogStore()
	manager, _ := projectTestManager(store)
	operator := catalogOperator()
	operator.DepartmentID = validationItemID

	if _, err := manager.CreateProject(context.Background(), operator, validCreateCommand()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
	if store.transactionCalls != 0 {
		t.Fatalf("transaction calls = %d, want 0", store.transactionCalls)
	}
}

func TestCatalogCRUDLifecycle(t *testing.T) {
	store := newMemoryCatalogStore()
	manager, _ := projectTestManager(store)
	operator := catalogOperator()
	operator.Permissions = []string{
		contractauthz.PermissionAppointmentCreate,
		contractauthz.PermissionAppointmentRead,
		contractauthz.PermissionAppointmentUpdate,
	}
	ctx := context.Background()

	created, err := manager.CreateProject(ctx, operator, validCreateCommand())
	if err != nil {
		t.Fatal(err)
	}
	got, err := manager.GetProject(ctx, operator, created.ItemID)
	if err != nil || got.ItemID != created.ItemID {
		t.Fatalf("get item=%#v err=%v", got, err)
	}
	page, err := manager.ListProjects(ctx, operator, ListProjectsQuery{Page: 1, PageSize: 10})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ItemID != created.ItemID {
		t.Fatalf("list result=%#v err=%v", page, err)
	}

	name := "Updated examination item"
	description := "Updated preparation description"
	updated, err := manager.UpdateProject(ctx, operator, UpdateProjectCommand{
		ItemID: created.ItemID, Name: &name, Description: &description,
		ExpectedVersion: 1, OperationID: "00000000-0000-0000-0000-000000000021",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.Description != description || updated.Version != 2 {
		t.Fatalf("unexpected updated item: %#v", updated)
	}

	disabled, err := manager.DisableProject(ctx, operator, ChangeProjectStatusCommand{
		ItemID: created.ItemID, ExpectedVersion: 2,
		OperationID: "00000000-0000-0000-0000-000000000022",
	})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Status != StatusDisabled || disabled.Version != 3 {
		t.Fatalf("unexpected disabled item: %#v", disabled)
	}

	blockedDescription := "must not be written"
	_, err = manager.UpdateProject(ctx, operator, UpdateProjectCommand{
		ItemID: created.ItemID, Description: &blockedDescription,
		ExpectedVersion: 3, OperationID: "00000000-0000-0000-0000-000000000023",
	})
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("disabled update error=%v, want ErrInvalidState", err)
	}

	enabledCommand := ChangeProjectStatusCommand{
		ItemID: created.ItemID, ExpectedVersion: 3,
		OperationID: "00000000-0000-0000-0000-000000000024",
	}
	enabled, err := manager.EnableProject(ctx, operator, enabledCommand)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := manager.EnableProject(ctx, operator, enabledCommand)
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Status != StatusActive || enabled.Version != 4 || replayed.Version != enabled.Version {
		t.Fatalf("enabled=%#v replayed=%#v", enabled, replayed)
	}
	if store.updateCalls != 1 || store.statusCalls != 2 {
		t.Fatalf("update calls=%d status calls=%d, want 1/2", store.updateCalls, store.statusCalls)
	}
}

func TestCatalogRejectsStaleVersion(t *testing.T) {
	store := newMemoryCatalogStore()
	manager, _ := projectTestManager(store)
	operator := catalogOperator()
	operator.Permissions = append(operator.Permissions, contractauthz.PermissionAppointmentUpdate)
	created, err := manager.CreateProject(context.Background(), operator, validCreateCommand())
	if err != nil {
		t.Fatal(err)
	}
	name := "stale update"
	_, err = manager.UpdateProject(context.Background(), operator, UpdateProjectCommand{
		ItemID: created.ItemID, Name: &name, ExpectedVersion: 99,
		OperationID: "00000000-0000-0000-0000-000000000025",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("error=%v, want ErrVersionConflict", err)
	}
}

func validCreateCommand() CreateProjectCommand {
	return CreateProjectCommand{
		OwnerDepartmentID: validationDepartmentID,
		Name:              "腹部 CT",
		Description:       "检查前禁食",
		OperationID:       validationOperationID,
		RequestID:         "request-create",
	}
}

func catalogOperator() authn.Principal {
	return authn.Principal{
		AccountID:    catalogOperatorID,
		AccountType:  authn.AccountTypeStaff,
		Status:       authn.AccountStatusActive,
		Roles:        []string{authn.RoleDepartmentDoctor},
		DepartmentID: validationDepartmentID,
		Permissions:  []string{contractauthz.PermissionAppointmentCreate},
	}
}

type memoryCatalogStore struct {
	operations       map[string]ProjectOperation
	items            map[string]ExaminationItem
	item             ExaminationItem
	change           *ProjectChange
	createCalls      int
	updateCalls      int
	statusCalls      int
	transactionCalls int
}

func newMemoryCatalogStore() *memoryCatalogStore {
	return &memoryCatalogStore{
		operations: make(map[string]ProjectOperation),
		items:      make(map[string]ExaminationItem),
	}
}

func (s *memoryCatalogStore) GetItem(_ context.Context, itemID string) (ExaminationItem, error) {
	item, exists := s.items[itemID]
	if !exists {
		return ExaminationItem{}, ErrNotFound
	}
	return item, nil
}

func (s *memoryCatalogStore) ListItems(_ context.Context, filter ProjectListFilter) ([]ExaminationItem, int64, error) {
	items := make([]ExaminationItem, 0)
	for _, item := range s.items {
		if filter.OwnerDepartmentID != "" && item.OwnerDepartmentID != filter.OwnerDepartmentID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].ItemID < items[j].ItemID
		}
		return items[i].Name < items[j].Name
	})
	total := int64(len(items))
	start := filter.Offset
	if start > total {
		start = total
	}
	end := start + filter.Limit
	if end > total {
		end = total
	}
	return items[int(start):int(end)], total, nil
}

func (s *memoryCatalogStore) WithinProjectTransaction(ctx context.Context, fn func(ProjectTxStore) error) error {
	s.transactionCalls++
	return fn(s)
}

func (s *memoryCatalogStore) FindOperation(_ context.Context, operationID string) (ProjectOperation, bool, error) {
	operation, exists := s.operations[operationID]
	return operation, exists, nil
}

func (s *memoryCatalogStore) GetItemForUpdate(_ context.Context, itemID string) (ExaminationItem, error) {
	return s.GetItem(context.Background(), itemID)
}

func (s *memoryCatalogStore) CreateItem(_ context.Context, item ExaminationItem) error {
	s.createCalls++
	s.item = item
	s.items[item.ItemID] = item
	return nil
}

func (s *memoryCatalogStore) UpdateItem(_ context.Context, item ExaminationItem, expectedVersion int64) error {
	current, exists := s.items[item.ItemID]
	if !exists {
		return ErrNotFound
	}
	if current.Version != expectedVersion {
		return ErrVersionConflict
	}
	s.updateCalls++
	s.items[item.ItemID] = item
	return nil
}

func (s *memoryCatalogStore) SetItemStatus(_ context.Context, item ExaminationItem, expectedVersion int64) error {
	current, exists := s.items[item.ItemID]
	if !exists {
		return ErrNotFound
	}
	if current.Version != expectedVersion {
		return ErrVersionConflict
	}
	s.statusCalls++
	s.items[item.ItemID] = item
	return nil
}

func (s *memoryCatalogStore) RecordChange(_ context.Context, change ProjectChange) error {
	s.change = &change
	s.operations[change.OperationID] = ProjectOperation{
		OperatorAccountID:  change.OperatorAccountID,
		ItemID:             change.ItemID,
		Action:             change.Action,
		RequestFingerprint: change.RequestFingerprint,
		Result:             change.After,
	}
	return nil
}
