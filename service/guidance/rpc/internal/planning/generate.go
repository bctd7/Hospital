package planning

import (
	"context"
	"time"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/projectconfiguration"

	"github.com/google/uuid"
)

// Generate 根据候选日期、可预约窗口和已确认规则生成最多三个宏观方案。
func (m *Manager) Generate(ctx context.Context, patient authn.Principal, command GenerateCommand) ([]Plan, error) {
	if err := requirePatient(patient); err != nil {
		return nil, err
	}
	itemIDs, availability, err := normalizeGenerate(command)
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
	allowedSlots := make(map[string]struct{}, len(availability)*2)
	for _, value := range availability {
		for _, session := range value.Sessions {
			slot := value.ServiceDate + ":" + session
			allowedSlots[slot] = struct{}{}
		}
	}
	activeBookings, err := m.appointment.ListMyBookings(ctx, "active")
	if err != nil {
		return nil, err
	}
	completedBookings, err := m.appointment.ListMyBookings(ctx, "completed")
	if err != nil {
		return nil, err
	}
	occupied := make(map[string]struct{}, len(activeBookings))
	for _, booking := range activeBookings {
		occupied[booking.ItemID+":"+booking.ServiceDate+":"+booking.Session] = struct{}{}
	}
	selectedItems := make(map[string]struct{}, len(itemIDs))
	for _, itemID := range itemIDs {
		selectedItems[itemID] = struct{}{}
	}
	allowedDates := make(map[string]struct{}, len(availability))
	for _, value := range availability {
		allowedDates[value.ServiceDate] = struct{}{}
	}
	existingBookings := append(append([]Booking(nil), activeBookings...), completedBookings...)
	precedenceBounds := existingPrecedenceBounds(itemIDs, selectedItems, rules, existingBookings, allowedDates)
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
			_, allowed := allowedSlots[option.ServiceDate+":"+option.Session]
			_, duplicate := occupied[itemID+":"+option.ServiceDate+":"+option.Session]
			if allowed && !duplicate && option.RemainingCapacity > 0 && precedenceBounds[itemID].allows(optionSlot(option)) {
				filtered = append(filtered, option)
			}
		}
		if len(filtered) == 0 {
			return nil, ErrNoPlan
		}
		projects[itemID], configurations[itemID], options[itemID] = project, configuration, filtered
	}

	travelEstimates := m.estimateBuildingTravel(ctx, options)
	candidates := searchBestPlansWithTravel(itemIDs, options, rules, configurations, travelEstimates)
	if len(candidates) == 0 {
		return nil, ErrNoPlan
	}
	requestFingerprint := fingerprint(command)
	plans := make([]Plan, 0, len(candidates))
	for variant, candidate := range candidates {
		items := make([]PlanItem, 0, len(candidate.choices))
		for _, selected := range candidate.choices {
			itemID, choice := selected.itemID, selected.option
			items = append(items, PlanItem{ItemID: itemID, ItemName: projects[itemID].Name, RoomID: choice.RoomID, RoomDisplayName: choice.RoomDisplayName, CampusID: choice.CampusID, Building: choice.Building, FloorNumber: choice.FloorNumber, RoomNumber: choice.RoomNumber, ServiceDate: choice.ServiceDate, Session: choice.Session, EstimatedDurationMinutes: choice.EstimatedDurationMinutes, PlannedStartTime: formatClockMinutes(selected.plannedStartMinutes), PlannedEndTime: formatClockMinutes(selected.plannedEndMinutes), TravelMinutes: selected.travelMinutes, TravelTimeEstimated: selected.travelEstimated, Reason: planReason(itemID, configurations[itemID], rules, items)})
		}
		plan := Plan{PlanID: uuid.NewString(), PatientAccountID: patient.AccountID, Title: planTitle(variant), Summary: planSummary(items), Items: items, ExpiresAt: time.Now().UTC().Add(30 * time.Minute)}
		if err := m.store.SavePlan(ctx, plan, requestFingerprint); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, nil
}
