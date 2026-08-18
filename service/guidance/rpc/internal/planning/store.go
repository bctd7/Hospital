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

// TravelEstimator 只提供智能预约所需的楼栋间步行分钟数，规划器不依赖地图供应商协议。
type TravelEstimator interface {
	EstimateWalkingMinutes(ctx context.Context, city, originKeyword, destinationKeyword string) (int32, error)
}
