package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hospital/service/guidance/rpc/internal/planning"
)

func (s *Store) SavePlan(ctx context.Context, plan planning.Plan, fingerprint string) error {
	payload, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("encode smart appointment plan: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO guidance_smart_appointment_plans
    (plan_id, patient_account_id, request_fingerprint, plan_payload, expires_at, created_at)
VALUES (?, ?, ?, ?, ?, ?)`, plan.PlanID, plan.PatientAccountID, fingerprint, payload, plan.ExpiresAt, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("save smart appointment plan: %w", err)
	}
	return nil
}

func (s *Store) GetPlan(ctx context.Context, planID, patientAccountID string) (planning.Plan, error) {
	var payload []byte
	var bookingIDsPayload []byte
	var confirmedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT plan_payload, confirmed_booking_ids, confirmed_at
FROM guidance_smart_appointment_plans WHERE plan_id = ? AND patient_account_id = ?`, planID, patientAccountID).Scan(&payload, &bookingIDsPayload, &confirmedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return planning.Plan{}, planning.ErrNotFound
	}
	if err != nil {
		return planning.Plan{}, fmt.Errorf("get smart appointment plan: %w", err)
	}
	var plan planning.Plan
	if err := json.Unmarshal(payload, &plan); err != nil {
		return planning.Plan{}, fmt.Errorf("decode smart appointment plan: %w", err)
	}
	if confirmedAt.Valid {
		plan.ConfirmedAt = &confirmedAt.Time
		if err := json.Unmarshal(bookingIDsPayload, &plan.BookingIDs); err != nil {
			return planning.Plan{}, fmt.Errorf("decode confirmed smart appointment bookings: %w", err)
		}
	}
	return plan, nil
}

func (s *Store) MarkPlanConfirmed(ctx context.Context, planID, patientAccountID string, bookingIDs []string) error {
	payload, err := json.Marshal(bookingIDs)
	if err != nil {
		return fmt.Errorf("encode confirmed smart appointment bookings: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE guidance_smart_appointment_plans
SET confirmed_booking_ids = CASE WHEN confirmed_at IS NULL THEN ? ELSE confirmed_booking_ids END,
    confirmed_at = COALESCE(confirmed_at, ?)
WHERE plan_id = ? AND patient_account_id = ?`, payload, time.Now().UTC(), planID, patientAccountID)
	if err != nil {
		return fmt.Errorf("confirm smart appointment plan: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read smart plan confirmation result: %w", err)
	}
	if affected != 1 {
		return planning.ErrNotFound
	}
	return nil
}

var _ planning.Store = (*Store)(nil)
