package logic

import (
	"errors"

	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/planning"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func planningRPCError(err error) error {
	switch {
	case errors.Is(err, planning.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid planning request")
	case errors.Is(err, planning.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, planning.ErrNotFound):
		return status.Error(codes.NotFound, "planning data not found")
	case errors.Is(err, planning.ErrNoPlan):
		return planningStatus(codes.FailedPrecondition, "NO_SMART_APPOINTMENT_PLAN", "当前选择无法生成预约方案，请检查已有预约，或重新选择日期和时段")
	case errors.Is(err, planning.ErrExistingBooking):
		return planningStatus(codes.FailedPrecondition, "SMART_APPOINTMENT_EXISTING_BOOKING", "当前方案与已有预约重复，请检查我的预约，或重新选择日期和时段")
	case errors.Is(err, planning.ErrConflict):
		return planningStatus(codes.Aborted, "SMART_APPOINTMENT_PLAN_CONFLICT", "预约条件已经变化，请重新生成方案")
	case errors.Is(err, planning.ErrUnavailable):
		return status.Error(codes.Unavailable, "planning dependency unavailable")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func planningStatus(code codes.Code, reason, message string) error {
	value, err := status.New(code, message).WithDetails(&errdetails.ErrorInfo{Reason: reason, Domain: "hospital.guidance"})
	if err != nil {
		return status.Error(code, message)
	}
	return value.Err()
}

func smartPlanResponse(value planning.Plan) *guidancev1.SmartAppointmentPlan {
	items := make([]*guidancev1.SmartAppointmentPlanItem, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, planItemResponse(item))
	}
	return &guidancev1.SmartAppointmentPlan{PlanId: value.PlanID, Title: value.Title, Summary: value.Summary, Items: items, ExpiresAt: formatGuidanceTime(value.ExpiresAt)}
}

func planItemResponse(value planning.PlanItem) *guidancev1.SmartAppointmentPlanItem {
	return &guidancev1.SmartAppointmentPlanItem{ItemId: value.ItemID, ItemName: value.ItemName, RoomId: value.RoomID, RoomDisplayName: value.RoomDisplayName, CampusId: value.CampusID, Building: value.Building, FloorNumber: value.FloorNumber, RoomNumber: value.RoomNumber, ServiceDate: value.ServiceDate, Session: value.Session, EstimatedDurationMinutes: value.EstimatedDurationMinutes, Reason: value.Reason, PlannedStartTime: value.PlannedStartTime, PlannedEndTime: value.PlannedEndTime, TravelMinutes: value.TravelMinutes, TravelTimeEstimated: value.TravelTimeEstimated}
}
