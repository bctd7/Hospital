package staff

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
)

const (
	projectOperatorID      = "00000000-0000-0000-0000-000000000020"
	validationDepartmentID = "00000000-0000-0000-0000-000000000010"
	validationItemID       = "00000000-0000-0000-0000-000000000011"
	validationOperationID  = "00000000-0000-0000-0000-000000000012"
)

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

func TestManagerRequiresProjectWriteStore(t *testing.T) {
	if _, err := NewManager(nil, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("expected nil project store to be rejected")
	}
	if manager, _ := projectTestManager(stubStore{}); manager == nil {
		t.Fatal("expected staff manager")
	}
}

func projectTestManager(store ProjectWriteStore) (*Manager, error) {
	return &Manager{projectWrites: store}, nil
}

func TestCreatePersistsActiveItemAndAudit(t *testing.T) {
	store := newMemoryProjectStore()
	manager, err := projectTestManager(store)
	if err != nil {
		t.Fatal(err)
	}

	created, err := manager.CreateProject(context.Background(), projectOperator(), staffinput.CreateProject{
		OwnerDepartmentID:        " " + validationDepartmentID + " ",
		Name:                     " 腹部 CT ",
		Description:              " 检查前禁食。\n可少量饮水。 ",
		EstimatedDurationMinutes: 30,
		OperationID:              validationOperationID,
		RequestID:                " request-create ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ItemID == "" || created.OwnerDepartmentID != validationDepartmentID ||
		created.Name != "腹部 CT" || created.Description != "检查前禁食。\n可少量饮水。" || created.EstimatedDurationMinutes != 30 ||
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
	store := newMemoryProjectStore()
	manager, _ := projectTestManager(store)
	command := validCreateCommand()

	created, err := manager.CreateProject(context.Background(), projectOperator(), command)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := manager.CreateProject(context.Background(), projectOperator(), command)
	if err != nil {
		t.Fatal(err)
	}
	if replayed.ItemID != created.ItemID || store.createCalls != 1 {
		t.Fatalf("replay=%#v created=%#v createCalls=%d", replayed, created, store.createCalls)
	}
}

func TestCreateRejectsOperationReuseWithDifferentRequest(t *testing.T) {
	store := newMemoryProjectStore()
	manager, _ := projectTestManager(store)
	command := validCreateCommand()
	if _, err := manager.CreateProject(context.Background(), projectOperator(), command); err != nil {
		t.Fatal(err)
	}
	command.Description = "不同描述"
	if _, err := manager.CreateProject(context.Background(), projectOperator(), command); !errors.Is(err, ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if store.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", store.createCalls)
	}
}

func TestCreateRejectsCrossDepartmentOperator(t *testing.T) {
	store := newMemoryProjectStore()
	manager, _ := projectTestManager(store)
	operator := projectOperator()
	operator.DepartmentID = validationItemID

	if _, err := manager.CreateProject(context.Background(), operator, validCreateCommand()); !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
	if store.transactionCalls != 0 {
		t.Fatalf("transaction calls = %d, want 0", store.transactionCalls)
	}
}

func TestProjectCRUDLifecycle(t *testing.T) {
	store := newMemoryProjectStore()
	manager, _ := projectTestManager(store)
	operator := projectOperator()
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
	name := "Updated examination item"
	description := "Updated preparation description"
	duration := int32(45)
	updated, err := manager.UpdateProject(ctx, operator, staffinput.UpdateProject{
		ItemID: created.ItemID, Name: &name, Description: &description, EstimatedDurationMinutes: &duration,
		ExpectedVersion: 1, OperationID: "00000000-0000-0000-0000-000000000021",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != name || updated.Description != description || updated.EstimatedDurationMinutes != duration || updated.Version != 2 {
		t.Fatalf("unexpected updated item: %#v", updated)
	}

	disabled, err := manager.DisableProject(ctx, operator, staffinput.ChangeProjectStatus{
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
	_, err = manager.UpdateProject(ctx, operator, staffinput.UpdateProject{
		ItemID: created.ItemID, Description: &blockedDescription,
		ExpectedVersion: 3, OperationID: "00000000-0000-0000-0000-000000000023",
	})
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("disabled update error=%v, want ErrInvalidState", err)
	}

	enabledCommand := staffinput.ChangeProjectStatus{
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

func TestProjectRejectsStaleVersion(t *testing.T) {
	store := newMemoryProjectStore()
	manager, _ := projectTestManager(store)
	operator := projectOperator()
	operator.Permissions = append(operator.Permissions, contractauthz.PermissionAppointmentUpdate)
	created, err := manager.CreateProject(context.Background(), operator, validCreateCommand())
	if err != nil {
		t.Fatal(err)
	}
	name := "stale update"
	_, err = manager.UpdateProject(context.Background(), operator, staffinput.UpdateProject{
		ItemID: created.ItemID, Name: &name, ExpectedVersion: 99,
		OperationID: "00000000-0000-0000-0000-000000000025",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("error=%v, want ErrVersionConflict", err)
	}
}

func TestProjectReportTemplateUsesIndependentVersionAndIdempotency(t *testing.T) {
	store := newMemoryProjectStore()
	manager, _ := projectTestManager(store)
	operator := projectOperator()
	operator.Permissions = append(operator.Permissions, contractauthz.PermissionAppointmentUpdate)
	created, err := manager.CreateProject(context.Background(), operator, validCreateCommand())
	if err != nil {
		t.Fatal(err)
	}
	command := staffinput.SaveReportTemplate{
		ItemID:                  created.ItemID,
		Template:                ReportContent{ObjectiveFindings: "模板所见", Impression: "模板结论"},
		ExpectedTemplateVersion: 0,
		Operation:               staffinput.Operation{OperationID: "00000000-0000-0000-0000-000000000026"},
	}
	saved, err := manager.SaveProjectReportTemplate(context.Background(), operator, command)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := manager.SaveProjectReportTemplate(context.Background(), operator, command)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ReportTemplate.Version != 1 || replayed.ReportTemplate.Version != 1 || saved.Version != created.Version {
		t.Fatalf("saved=%#v replayed=%#v", saved, replayed)
	}
	command.OperationID = "00000000-0000-0000-0000-000000000027"
	if _, err := manager.SaveProjectReportTemplate(context.Background(), operator, command); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("error=%v, want ErrVersionConflict", err)
	}
}

func validCreateCommand() staffinput.CreateProject {
	return staffinput.CreateProject{
		OwnerDepartmentID:        validationDepartmentID,
		Name:                     "腹部 CT",
		Description:              "检查前禁食",
		EstimatedDurationMinutes: 30,
		OperationID:              validationOperationID,
		RequestID:                "request-create",
	}
}

func projectOperator() authn.Principal {
	return authn.Principal{
		AccountID:    projectOperatorID,
		AccountType:  authn.AccountTypeStaff,
		Status:       authn.AccountStatusActive,
		Roles:        []string{authn.RoleDepartmentDoctor},
		DepartmentID: validationDepartmentID,
		Permissions:  []string{contractauthz.PermissionAppointmentCreate},
	}
}

type memoryProjectStore struct {
	operations       map[string]ProjectOperation
	items            map[string]ExaminationItem
	item             ExaminationItem
	change           *ProjectChange
	createCalls      int
	updateCalls      int
	statusCalls      int
	transactionCalls int
}

func newMemoryProjectStore() *memoryProjectStore {
	return &memoryProjectStore{
		operations: make(map[string]ProjectOperation),
		items:      make(map[string]ExaminationItem),
	}
}

func (s *memoryProjectStore) GetItem(_ context.Context, itemID string) (ExaminationItem, error) {
	item, exists := s.items[itemID]
	if !exists {
		return ExaminationItem{}, ErrNotFound
	}
	return item, nil
}

func (s *memoryProjectStore) ListItems(_ context.Context, filter ProjectListFilter) ([]ExaminationItem, int64, error) {
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

func (s *memoryProjectStore) WithinProjectTransaction(ctx context.Context, fn func(ProjectTxStore) error) error {
	s.transactionCalls++
	return fn(s)
}

func (s *memoryProjectStore) FindOperation(_ context.Context, operationID string) (ProjectOperation, bool, error) {
	operation, exists := s.operations[operationID]
	return operation, exists, nil
}

func (s *memoryProjectStore) GetItemForUpdate(_ context.Context, itemID string) (ExaminationItem, error) {
	return s.GetItem(context.Background(), itemID)
}

func (s *memoryProjectStore) CreateItem(_ context.Context, item ExaminationItem) error {
	s.createCalls++
	s.item = item
	s.items[item.ItemID] = item
	return nil
}

func (s *memoryProjectStore) UpdateItem(_ context.Context, item ExaminationItem, expectedVersion int64) error {
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

func (s *memoryProjectStore) UpdateItemReportTemplate(_ context.Context, item ExaminationItem, expectedTemplateVersion int64) error {
	current, exists := s.items[item.ItemID]
	if !exists {
		return ErrNotFound
	}
	if current.ReportTemplate.Version != expectedTemplateVersion {
		return ErrVersionConflict
	}
	s.items[item.ItemID] = item
	return nil
}

func (s *memoryProjectStore) SetItemStatus(_ context.Context, item ExaminationItem, expectedVersion int64) error {
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

func (s *memoryProjectStore) RecordChange(_ context.Context, change ProjectChange) error {
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

func (s *memoryProjectStore) FindBookingOperation(context.Context, string) (BookingOperation, bool, error) {
	return BookingOperation{}, false, nil
}

func (s *memoryProjectStore) RecordBookingOperation(context.Context, BookingOperationChange) error {
	return nil
}

func (s *memoryProjectStore) ListBookingsForUpdate(context.Context, BookingListFilter) ([]Booking, error) {
	return nil, nil
}

func (s *memoryProjectStore) ClaimPatientItemSession(context.Context, string, string, time.Time, Session, string) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) ReleasePatientItemSession(context.Context, string) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) ConsumePatientWeeklyQuota(context.Context, string, time.Time, int64, time.Time) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) LockBookingSelection(context.Context, string, string, time.Time, Session) (BookingSelection, error) {
	return BookingSelection{}, ErrNotImplemented
}

func (s *memoryProjectStore) FindDateCapacityForUpdate(context.Context, string, time.Time, Session) (DateCapacity, bool, error) {
	return DateCapacity{}, false, nil
}

func (s *memoryProjectStore) CreateDateCapacity(context.Context, DateCapacity) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) IncreaseOccupiedCapacity(context.Context, string) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) DecreaseOccupiedCapacity(context.Context, string) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) UpdateDateCapacity(context.Context, string, int64, string, int64) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) GetBookingForUpdate(context.Context, string) (Booking, error) {
	return Booking{}, ErrNotImplemented
}

func (s *memoryProjectStore) CreateBooking(context.Context, Booking) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) MarkBookingNoShow(context.Context, Booking, int64) error {
	return ErrNotImplemented
}

func (s *memoryProjectStore) DeleteBooking(context.Context, string) error {
	return ErrNotImplemented
}
