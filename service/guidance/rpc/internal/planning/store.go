package planning

import (
	"context"

	"hospital/service/guidance/rpc/internal/projectconfiguration"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type Store interface {
	GetConfiguration(ctx context.Context, itemID string) (projectconfiguration.Configuration, error)
	ListAll(ctx context.Context) ([]precedence.Rule, error)
	SavePlan(ctx context.Context, plan Plan, fingerprint string) error
	GetPlan(ctx context.Context, planID, patientAccountID string) (Plan, error)
	MarkPlanConfirmed(ctx context.Context, planID, patientAccountID string, bookingIDs []string) error
}

type Appointment interface {
	ResolveProject(ctx context.Context, itemID string) (projectconfiguration.Project, error)
	ListOptions(ctx context.Context, itemID string) ([]Option, error)
	ListMyBookings(ctx context.Context, view string) ([]Booking, error)
	CreateBookingBatch(ctx context.Context, command ConfirmCommand, items []PlanItem) ([]string, error)
}
