package planning

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/projectconfiguration"
	descriptionrules "hospital/service/guidance/rpc/internal/rules/description"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type planningStoreStub struct {
	configurations map[string]projectconfiguration.Configuration
	rules          []precedence.Rule
	plans          map[string]Plan
}

func (s *planningStoreStub) GetConfiguration(_ context.Context, itemID string) (projectconfiguration.Configuration, error) {
	value, ok := s.configurations[itemID]
	if !ok {
		return projectconfiguration.Configuration{}, ErrNotFound
	}
	return value, nil
}
func (s *planningStoreStub) ListAll(context.Context) ([]precedence.Rule, error) {
	return append([]precedence.Rule(nil), s.rules...), nil
}
func (s *planningStoreStub) SavePlan(_ context.Context, plan Plan, _ string) error {
	s.plans[plan.PlanID] = plan
	return nil
}
func (s *planningStoreStub) GetPlan(_ context.Context, planID, patientID string) (Plan, error) {
	value, ok := s.plans[planID]
	if !ok || value.PatientAccountID != patientID {
		return Plan{}, ErrNotFound
	}
	return value, nil
}
func (s *planningStoreStub) MarkPlanConfirmed(_ context.Context, planID, patientID string, bookingIDs []string) error {
	value, ok := s.plans[planID]
	if !ok || value.PatientAccountID != patientID {
		return ErrNotFound
	}
	now := time.Now().UTC()
	value.ConfirmedAt = &now
	value.BookingIDs = append([]string(nil), bookingIDs...)
	s.plans[planID] = value
	return nil
}

type planningAppointmentStub struct {
	projects    map[string]projectconfiguration.Project
	options     map[string][]Option
	bookings    map[string][]Booking
	batchCalls  int
	batchResult []string
}

func (s *planningAppointmentStub) ResolveProject(_ context.Context, itemID string) (projectconfiguration.Project, error) {
	value, ok := s.projects[itemID]
	if !ok {
		return projectconfiguration.Project{}, ErrNotFound
	}
	return value, nil
}
func (s *planningAppointmentStub) ListOptions(_ context.Context, itemID string) ([]Option, error) {
	return append([]Option(nil), s.options[itemID]...), nil
}
func (s *planningAppointmentStub) ListMyBookings(_ context.Context, view string) ([]Booking, error) {
	return append([]Booking(nil), s.bookings[view]...), nil
}
func (s *planningAppointmentStub) CreateBookingBatch(context.Context, ConfirmCommand, []PlanItem) ([]string, error) {
	s.batchCalls++
	return append([]string(nil), s.batchResult...), nil
}

func TestGeneratePreservesDirectOrderAcrossActualBookingSlots(t *testing.T) {
	first, second := uuid.NewString(), uuid.NewString()
	store := &planningStoreStub{
		configurations: map[string]projectconfiguration.Configuration{
			first:  {ItemID: first, PreparationRules: []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeNoWater}}},
			second: {ItemID: second, PreparationRules: []descriptionrules.Rule{{RuleType: descriptionrules.RuleTypeDrinkWater}}},
		},
		rules: []precedence.Rule{{PredecessorItemID: first, SuccessorItemID: second}}, plans: map[string]Plan{},
	}
	appointment := &planningAppointmentStub{
		projects: map[string]projectconfiguration.Project{
			first: {ItemID: first, Name: "禁水检查", Status: "active"}, second: {ItemID: second, Name: "饮水检查", Status: "active"},
		},
		options: map[string][]Option{
			first: {{ItemID: first, RoomID: uuid.NewString(), ServiceDate: "2026-08-18", Session: "afternoon", RemainingCapacity: 2}},
			second: {
				{ItemID: second, RoomID: uuid.NewString(), ServiceDate: "2026-08-17", Session: "morning", RemainingCapacity: 4},
				{ItemID: second, RoomID: uuid.NewString(), ServiceDate: "2026-08-18", Session: "afternoon", RemainingCapacity: 3},
			},
		}, bookings: map[string][]Booking{},
	}
	manager, err := NewManager(store, appointment, nil)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	plans, err := manager.Generate(context.Background(), patientPrincipal(), GenerateCommand{ItemIDs: []string{second, first}, CandidateDates: []string{"2026-08-17", "2026-08-18"}})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(plans) == 0 || len(plans[0].Items) != 2 {
		t.Fatalf("unexpected plans: %#v", plans)
	}
	got := plans[0].Items
	if got[0].ItemID != first || got[1].ItemID != second {
		t.Fatalf("direct order not preserved: %#v", got)
	}
	if got[1].ServiceDate < got[0].ServiceDate || (got[1].ServiceDate == got[0].ServiceDate && sessionRank(got[1].Session) < sessionRank(got[0].Session)) {
		t.Fatalf("successor booked before predecessor: %#v", got)
	}
}

func TestGenerateAcceptsMorningOnFirstCandidateDate(t *testing.T) {
	itemID := uuid.NewString()
	store := &planningStoreStub{
		configurations: map[string]projectconfiguration.Configuration{itemID: {ItemID: itemID}},
		plans:          map[string]Plan{},
	}
	appointment := &planningAppointmentStub{
		projects: map[string]projectconfiguration.Project{itemID: {ItemID: itemID, Name: "当天检查", Status: "active"}},
		options: map[string][]Option{itemID: {{
			ItemID: itemID, RoomID: uuid.NewString(), ServiceDate: "2026-08-18", Session: "morning", RemainingCapacity: 1,
		}}},
		bookings: map[string][]Booking{},
	}
	manager, _ := NewManager(store, appointment, nil)
	plans, err := manager.Generate(context.Background(), patientPrincipal(), GenerateCommand{ItemIDs: []string{itemID}, CandidateDates: []string{"2026-08-18"}})
	if err != nil || len(plans) != 1 {
		t.Fatalf("expected a plan on the first candidate morning, plans=%#v err=%v", plans, err)
	}
}

func TestGenerateHonorsPerDateSessionSelection(t *testing.T) {
	itemID := uuid.NewString()
	store := &planningStoreStub{configurations: map[string]projectconfiguration.Configuration{itemID: {ItemID: itemID}}, plans: map[string]Plan{}}
	appointment := &planningAppointmentStub{
		projects: map[string]projectconfiguration.Project{itemID: {ItemID: itemID, Name: "可选时段检查", Status: "active"}},
		options: map[string][]Option{itemID: {
			{ItemID: itemID, RoomID: uuid.NewString(), ServiceDate: "2026-08-18", Session: "morning", RemainingCapacity: 1},
			{ItemID: itemID, RoomID: uuid.NewString(), ServiceDate: "2026-08-18", Session: "afternoon", RemainingCapacity: 1},
		}}, bookings: map[string][]Booking{},
	}
	manager, _ := NewManager(store, appointment, nil)
	plans, err := manager.Generate(context.Background(), patientPrincipal(), GenerateCommand{ItemIDs: []string{itemID}, CandidateAvailability: []CandidateAvailability{{ServiceDate: "2026-08-18", Sessions: []string{"afternoon"}}}})
	if err != nil || len(plans) != 1 || plans[0].Items[0].Session != "afternoon" {
		t.Fatalf("expected afternoon-only plan, plans=%#v err=%v", plans, err)
	}
}

func TestGenerateRejectsDuplicateActiveBookingForSameItemAndSession(t *testing.T) {
	itemID := uuid.NewString()
	store := &planningStoreStub{configurations: map[string]projectconfiguration.Configuration{itemID: {ItemID: itemID}}, plans: map[string]Plan{}}
	appointment := &planningAppointmentStub{
		projects: map[string]projectconfiguration.Project{itemID: {ItemID: itemID, Name: "重复检查", Status: "active"}},
		options:  map[string][]Option{itemID: {{ItemID: itemID, RoomID: uuid.NewString(), ServiceDate: "2026-08-18", Session: "morning", RemainingCapacity: 2}}},
		bookings: map[string][]Booking{"active": {{ItemID: itemID, ServiceDate: "2026-08-18", Session: "morning"}}},
	}
	manager, _ := NewManager(store, appointment, nil)
	_, err := manager.Generate(context.Background(), patientPrincipal(), GenerateCommand{ItemIDs: []string{itemID}, CandidateAvailability: []CandidateAvailability{{ServiceDate: "2026-08-18", Sessions: []string{"morning"}}}})
	if err != ErrNoPlan {
		t.Fatalf("expected ErrNoPlan for an existing active booking in the same item/date/session, got %v", err)
	}
}

func TestGenerateDoesNotOverbookSharedRoomCapacity(t *testing.T) {
	first, second := uuid.NewString(), uuid.NewString()
	roomID := uuid.NewString()
	store := &planningStoreStub{
		configurations: map[string]projectconfiguration.Configuration{first: {ItemID: first}, second: {ItemID: second}},
		plans:          map[string]Plan{},
	}
	appointment := &planningAppointmentStub{
		projects: map[string]projectconfiguration.Project{
			first: {ItemID: first, Name: "检查一", Status: "active"}, second: {ItemID: second, Name: "检查二", Status: "active"},
		},
		options: map[string][]Option{
			first:  {{ItemID: first, RoomID: roomID, ServiceDate: "2026-08-18", Session: "morning", RemainingCapacity: 1}},
			second: {{ItemID: second, RoomID: roomID, ServiceDate: "2026-08-18", Session: "morning", RemainingCapacity: 1}},
		},
		bookings: map[string][]Booking{},
	}
	manager, _ := NewManager(store, appointment, nil)
	if _, err := manager.Generate(context.Background(), patientPrincipal(), GenerateCommand{ItemIDs: []string{first, second}, CandidateDates: []string{"2026-08-18"}}); err != ErrNoPlan {
		t.Fatalf("expected ErrNoPlan when two projects share one remaining capacity, got %v", err)
	}
}

func TestConfirmReturnsStoredBookingsWithoutCreatingTwice(t *testing.T) {
	patient := patientPrincipal()
	planID, operationID, bookingID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	store := &planningStoreStub{configurations: map[string]projectconfiguration.Configuration{}, plans: map[string]Plan{
		planID: {PlanID: planID, PatientAccountID: patient.AccountID, ExpiresAt: time.Now().Add(time.Hour), Items: []PlanItem{{ItemID: uuid.NewString()}}},
	}}
	appointment := &planningAppointmentStub{projects: map[string]projectconfiguration.Project{}, options: map[string][]Option{}, bookings: map[string][]Booking{}, batchResult: []string{bookingID}}
	manager, _ := NewManager(store, appointment, nil)
	command := ConfirmCommand{PlanID: planID, OperationID: operationID}
	first, err := manager.Confirm(context.Background(), patient, command)
	if err != nil {
		t.Fatalf("first Confirm() error = %v", err)
	}
	second, err := manager.Confirm(context.Background(), patient, command)
	if err != nil {
		t.Fatalf("second Confirm() error = %v", err)
	}
	if appointment.batchCalls != 1 || len(first) != 1 || first[0] != bookingID || len(second) != 1 || second[0] != bookingID {
		t.Fatalf("confirmation is not idempotent: calls=%d first=%v second=%v", appointment.batchCalls, first, second)
	}
}

func patientPrincipal() authn.Principal {
	return authn.Principal{AccountID: uuid.NewString(), AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive}
}
