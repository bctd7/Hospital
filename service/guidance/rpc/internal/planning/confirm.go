package planning

import (
	"context"
	"time"

	"hospital/common/authn"
)

// Confirm 将一个仍有效的推荐方案交给 Appointment 原子创建整组预约。
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
