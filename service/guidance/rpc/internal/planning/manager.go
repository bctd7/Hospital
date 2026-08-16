package planning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/projectconfiguration"
	"hospital/service/guidance/rpc/internal/rules/precedence"
	"hospital/service/guidance/rpc/internal/rules/preparation"
)

var shanghai = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}()

type Manager struct {
	store       Store
	appointment Appointment
}

func NewManager(store Store, appointment Appointment) (*Manager, error) {
	if store == nil || appointment == nil {
		return nil, errors.New("planning dependencies are required")
	}
	return &Manager{store: store, appointment: appointment}, nil
}

func (m *Manager) Generate(ctx context.Context, patient authn.Principal, command GenerateCommand) ([]Plan, error) {
	if err := requirePatient(patient); err != nil {
		return nil, err
	}
	itemIDs, dates, err := normalizeGenerate(command)
	if err != nil {
		return nil, err
	}
	rules, err := m.store.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	projects := make(map[string]projectconfiguration.Project, len(itemIDs))
	configurations := make(map[string]projectconfiguration.Configuration, len(itemIDs))
	options := make(map[string][]Option, len(itemIDs))
	dateSet := make(map[string]struct{}, len(dates))
	for _, date := range dates {
		dateSet[date] = struct{}{}
	}
	for _, itemID := range itemIDs {
		project, loadErr := m.appointment.ResolveProject(ctx, itemID)
		if loadErr != nil || project.Status != "active" {
			return nil, ErrNoPlan
		}
		configuration, loadErr := m.store.GetConfiguration(ctx, itemID)
		if loadErr != nil {
			return nil, ErrNoPlan
		}
		available, loadErr := m.appointment.ListOptions(ctx, itemID)
		if loadErr != nil {
			return nil, loadErr
		}
		filtered := available[:0]
		for _, option := range available {
			if _, ok := dateSet[option.ServiceDate]; ok && option.RemainingCapacity > 0 {
				filtered = append(filtered, option)
			}
		}
		if len(filtered) == 0 {
			return nil, ErrNoPlan
		}
		projects[itemID], configurations[itemID], options[itemID] = project, configuration, filtered
	}
	ordered := orderItems(itemIDs, rules, configurations)
	fingerprint := fingerprint(command)
	plans := make([]Plan, 0, 3)
	seen := make(map[string]struct{})
	for variant := 0; variant < 3; variant++ {
		if variant >= len(dates) {
			break
		}
		items := make([]PlanItem, 0, len(ordered))
		minimumSlot := dates[variant] + ":0"
		reservedCapacity := make(map[string]int64)
		valid := true
		for _, itemID := range ordered {
			available := append([]Option(nil), options[itemID]...)
			sort.SliceStable(available, func(i, j int) bool {
				leftSlot, rightSlot := optionSlot(available[i]), optionSlot(available[j])
				if leftSlot != rightSlot {
					return leftSlot < rightSlot
				}
				if len(items) > 0 {
					previousBuilding := items[len(items)-1].Building
					leftSame, rightSame := available[i].Building == previousBuilding, available[j].Building == previousBuilding
					if leftSame != rightSame {
						return leftSame
					}
				}
				if available[i].Building != available[j].Building {
					return available[i].Building < available[j].Building
				}
				return available[i].RemainingCapacity > available[j].RemainingCapacity
			})
			choiceIndex := -1
			for index := range available {
				capacityKey := optionCapacityKey(available[index])
				if optionSlot(available[index]) >= minimumSlot && reservedCapacity[capacityKey] < available[index].RemainingCapacity {
					choiceIndex = index
					break
				}
			}
			if choiceIndex < 0 {
				valid = false
				break
			}
			choice := available[choiceIndex]
			reservedCapacity[optionCapacityKey(choice)]++
			minimumSlot = optionSlot(choice)
			items = append(items, PlanItem{ItemID: itemID, ItemName: projects[itemID].Name, RoomID: choice.RoomID, RoomDisplayName: choice.RoomDisplayName, CampusID: choice.CampusID, Building: choice.Building, FloorNumber: choice.FloorNumber, RoomNumber: choice.RoomNumber, ServiceDate: choice.ServiceDate, Session: choice.Session, EstimatedDurationMinutes: choice.EstimatedDurationMinutes, Reason: planReason(itemID, configurations[itemID], rules, items)})
		}
		if !valid {
			continue
		}
		keyBytes, _ := json.Marshal(items)
		key := string(keyBytes)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		expiresAt := time.Now().UTC().Add(30 * time.Minute)
		plan := Plan{PlanID: uuid.NewString(), PatientAccountID: patient.AccountID, Title: planTitle(variant), Summary: planSummary(items), Items: items, ExpiresAt: expiresAt}
		if err := m.store.SavePlan(ctx, plan, fingerprint); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	if len(plans) == 0 {
		return nil, ErrNoPlan
	}
	return plans, nil
}

func (m *Manager) Confirm(ctx context.Context, patient authn.Principal, command ConfirmCommand) ([]string, error) {
	if err := requirePatient(patient); err != nil {
		return nil, err
	}
	planID, err := normalizeUUID(command.PlanID)
	if err != nil {
		return nil, err
	}
	command.OperationID, err = normalizeUUID(command.OperationID)
	if err != nil {
		return nil, err
	}
	plan, err := m.store.GetPlan(ctx, planID, patient.AccountID)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(plan.ExpiresAt) {
		return nil, ErrConflict
	}
	if plan.ConfirmedAt != nil {
		return append([]string(nil), plan.BookingIDs...), nil
	}
	ids, err := m.appointment.CreateBookingBatch(ctx, command, plan.Items)
	if err != nil {
		return nil, err
	}
	if err := m.store.MarkPlanConfirmed(ctx, planID, patient.AccountID, ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (m *Manager) Today(ctx context.Context, patient authn.Principal) (TodayRecommendation, error) {
	if err := requirePatient(patient); err != nil {
		return TodayRecommendation{}, err
	}
	active, err := m.appointment.ListMyBookings(ctx, "active")
	if err != nil {
		return TodayRecommendation{}, err
	}
	completed, err := m.appointment.ListMyBookings(ctx, "completed")
	if err != nil {
		return TodayRecommendation{}, err
	}
	today := time.Now().In(shanghai).Format("2006-01-02")
	bookings := make([]Booking, 0)
	for _, value := range append(active, completed...) {
		if value.ServiceDate == today {
			bookings = append(bookings, value)
		}
	}
	if len(bookings) == 0 {
		return TodayRecommendation{ServiceDate: today, Stages: []Stage{}, UpdatedAt: time.Now().UTC()}, nil
	}
	itemIDs := make([]string, 0, len(bookings))
	configurations := make(map[string]projectconfiguration.Configuration)
	for _, booking := range bookings {
		itemIDs = append(itemIDs, booking.ItemID)
		if configuration, loadErr := m.store.GetConfiguration(ctx, booking.ItemID); loadErr == nil {
			configurations[booking.ItemID] = configuration
		}
	}
	rules, err := m.store.ListAll(ctx)
	if err != nil {
		return TodayRecommendation{}, err
	}
	order := orderItems(unique(itemIDs), rules, configurations)
	orderIndex := make(map[string]int, len(order))
	for index, itemID := range order {
		orderIndex[itemID] = index
	}
	sort.SliceStable(bookings, func(i, j int) bool {
		left, right := statusRank(bookings[i].Status), statusRank(bookings[j].Status)
		if left != right {
			return left < right
		}
		return orderIndex[bookings[i].ItemID] < orderIndex[bookings[j].ItemID]
	})
	groups := []struct {
		title, status, focus string
		statuses             map[string]bool
	}{
		{"正在检查", "in_progress", "请按现场工作人员指引完成当前检查。", map[string]bool{"in_progress": true}},
		{"正在叫号", "called", "请留意现场广播并尽快前往检查房间。", map[string]bool{"called": true}},
		{"候检与待到院", "waiting", "建议按以下顺序前往；同楼栋项目已尽量集中。", map[string]bool{"queued": true, "confirmed": true}},
		{"今日已完成", "completed", "以下项目已完成，不再参与后续顺序计算。", map[string]bool{"report_pending": true, "completed": true, "no_show": true}},
	}
	stages := make([]Stage, 0, len(groups))
	for _, group := range groups {
		items := make([]PlanItem, 0)
		for _, booking := range bookings {
			if group.statuses[booking.Status] {
				items = append(items, bookingPlanItem(booking))
			}
		}
		if len(items) > 0 {
			stages = append(stages, Stage{StageNo: int32(len(stages) + 1), Title: group.title, Status: group.status, Focus: group.focus, Items: items})
		}
	}
	return TodayRecommendation{ServiceDate: today, Stages: stages, UpdatedAt: time.Now().UTC()}, nil
}

func normalizeGenerate(command GenerateCommand) ([]string, []string, error) {
	if len(command.ItemIDs) == 0 || len(command.ItemIDs) > 10 || len(command.CandidateDates) == 0 || len(command.CandidateDates) > 14 {
		return nil, nil, ErrInvalid
	}
	items := unique(command.ItemIDs)
	for index, value := range items {
		normalized, err := normalizeUUID(value)
		if err != nil {
			return nil, nil, err
		}
		items[index] = normalized
	}
	dates := unique(command.CandidateDates)
	for _, value := range dates {
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return nil, nil, ErrInvalid
		}
	}
	sort.Strings(dates)
	return items, dates, nil
}

func requirePatient(principal authn.Principal) error {
	if principal.AccountID == "" || principal.Status != authn.AccountStatusActive {
		return ErrForbidden
	}
	if principal.AccountType != authn.AccountTypePatient && principal.AccountType != authn.AccountTypeStaff {
		return ErrForbidden
	}
	if _, err := uuid.Parse(principal.AccountID); err != nil {
		return ErrForbidden
	}
	return nil
}

func normalizeUUID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", ErrInvalid
	}
	return parsed.String(), nil
}
func unique(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if _, ok := seen[value]; !ok && value != "" {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
func fingerprint(value any) string {
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func orderItems(itemIDs []string, rules []precedence.Rule, configurations map[string]projectconfiguration.Configuration) []string {
	selected := make(map[string]bool, len(itemIDs))
	for _, id := range itemIDs {
		selected[id] = true
	}
	graph, indegree := map[string][]string{}, map[string]int{}
	for _, id := range itemIDs {
		indegree[id] = 0
	}
	for _, rule := range rules {
		if selected[rule.PredecessorItemID] && selected[rule.SuccessorItemID] {
			graph[rule.PredecessorItemID] = append(graph[rule.PredecessorItemID], rule.SuccessorItemID)
			indegree[rule.SuccessorItemID]++
		}
	}
	result := make([]string, 0, len(itemIDs))
	for len(result) < len(itemIDs) {
		ready := make([]string, 0)
		for _, id := range itemIDs {
			if indegree[id] == 0 && !contains(result, id) {
				ready = append(ready, id)
			}
		}
		if len(ready) == 0 {
			return append([]string(nil), itemIDs...)
		}
		sort.SliceStable(ready, func(i, j int) bool {
			left, right := preparationRank(configurations[ready[i]]), preparationRank(configurations[ready[j]])
			return left < right || (left == right && ready[i] < ready[j])
		})
		id := ready[0]
		result = append(result, id)
		for _, next := range graph[id] {
			indegree[next]--
		}
	}
	return result
}

func preparationRank(configuration projectconfiguration.Configuration) int {
	hasRestricted, hasDrink := false, false
	for _, rule := range configuration.PreparationRules {
		if rule.RuleType == preparation.RuleTypeFasting || rule.RuleType == preparation.RuleTypeNoWater {
			hasRestricted = true
		}
		if rule.RuleType == preparation.RuleTypeDrinkWater {
			hasDrink = true
		}
	}
	if hasRestricted {
		return 0
	}
	if hasDrink {
		return 2
	}
	return 1
}
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
func optionSlot(value Option) string {
	return value.ServiceDate + ":" + strconv.Itoa(sessionRank(value.Session))
}

func optionCapacityKey(value Option) string {
	return value.RoomID + ":" + value.ServiceDate + ":" + value.Session
}
func sessionRank(value string) int {
	if value == "morning" {
		return 0
	}
	return 1
}
func planTitle(variant int) string {
	return []string{"优先推荐", "备选方案一", "备选方案二"}[variant]
}
func planSummary(items []PlanItem) string {
	if len(items) == 0 {
		return ""
	}
	buildings := unique(func() []string {
		values := make([]string, 0, len(items))
		for _, item := range items {
			values = append(values, item.Building)
		}
		return values
	}())
	return "共安排" + strconv.Itoa(len(items)) + "项检查，涉及" + strings.Join(buildings, "、")
}
func planReason(itemID string, configuration projectconfiguration.Configuration, rules []precedence.Rule, previous []PlanItem) string {
	if preparationRank(configuration) == 0 {
		return "优先完成禁食或禁水项目"
	}
	if preparationRank(configuration) == 2 {
		return "安排在饮水准备状态满足后"
	}
	for _, rule := range rules {
		if rule.SuccessorItemID == itemID {
			if message := strings.TrimSpace(rule.PatientMessage); message != "" {
				return message
			}
			return "按项目先后关系安排"
		}
	}
	if len(previous) > 0 && previous[len(previous)-1].Building != "" {
		return "同楼栋项目尽量集中"
	}
	return "按可预约窗口安排"
}
func statusRank(status string) int {
	switch status {
	case "in_progress":
		return 0
	case "called":
		return 1
	case "queued":
		return 2
	case "confirmed":
		return 3
	default:
		return 4
	}
}
func bookingPlanItem(value Booking) PlanItem {
	return PlanItem{ItemID: value.ItemID, ItemName: value.ItemName, RoomID: value.RoomID, RoomDisplayName: value.RoomDisplayName, CampusID: value.CampusID, Building: value.Building, FloorNumber: value.FloorNumber, RoomNumber: value.RoomNumber, ServiceDate: value.ServiceDate, Session: value.Session, EstimatedDurationMinutes: value.EstimatedDurationMinutes, Reason: value.Status}
}
