package planning

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
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
	startingSlots := make([]string, 0, len(availability)*2)
	for _, value := range availability {
		for _, session := range value.Sessions {
			slot := value.ServiceDate + ":" + session
			allowedSlots[slot] = struct{}{}
			startingSlots = append(startingSlots, value.ServiceDate+":"+strconv.Itoa(sessionRank(session)))
		}
	}
	activeBookings, err := m.appointment.ListMyBookings(ctx, "active")
	if err != nil {
		return nil, err
	}
	occupied := make(map[string]struct{}, len(activeBookings))
	for _, booking := range activeBookings {
		occupied[booking.ItemID+":"+booking.ServiceDate+":"+booking.Session] = struct{}{}
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
			_, allowed := allowedSlots[option.ServiceDate+":"+option.Session]
			_, duplicate := occupied[itemID+":"+option.ServiceDate+":"+option.Session]
			if allowed && !duplicate && option.RemainingCapacity > 0 {
				filtered = append(filtered, option)
			}
		}
		if len(filtered) == 0 {
			return nil, ErrNoPlan
		}
		projects[itemID], configurations[itemID], options[itemID] = project, configuration, filtered
	}

	ordered := orderItems(itemIDs, rules, configurations)
	requestFingerprint := fingerprint(command)
	plans := make([]Plan, 0, 3)
	seen := make(map[string]struct{})
	for variant := 0; variant < 3 && variant < len(startingSlots); variant++ {
		items := make([]PlanItem, 0, len(ordered))
		minimumSlot := startingSlots[variant]
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
		plan := Plan{PlanID: uuid.NewString(), PatientAccountID: patient.AccountID, Title: planTitle(variant), Summary: planSummary(items), Items: items, ExpiresAt: time.Now().UTC().Add(30 * time.Minute)}
		if err := m.store.SavePlan(ctx, plan, requestFingerprint); err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	if len(plans) == 0 {
		return nil, ErrNoPlan
	}
	return plans, nil
}
