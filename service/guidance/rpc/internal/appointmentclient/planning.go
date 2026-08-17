package appointmentclient

import (
	"context"
	"fmt"

	"hospital/service/appointment/rpc/appointmentservice"
	"hospital/service/guidance/rpc/internal/planning"
	"hospital/service/guidance/rpc/internal/projectconfiguration"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Planning 负责智能预约和当日顺序所需的 Appointment 只读查询与批量预约。
type Planning struct {
	client appointmentservice.AppointmentService
}

func NewPlanning(client appointmentservice.AppointmentService) (*Planning, error) {
	if client == nil {
		return nil, fmt.Errorf("appointment client is required")
	}
	return &Planning{client: client}, nil
}

func (g *Planning) ResolveProject(ctx context.Context, itemID string) (projectconfiguration.Project, error) {
	value, err := g.client.GetExaminationItemReference(outgoingContext(ctx), &appointmentservice.GetExaminationItemRequest{ItemId: itemID})
	if err != nil {
		return projectconfiguration.Project{}, fmt.Errorf("resolve planning project: %w", err)
	}
	return configurationProject(value), nil
}

func (g *Planning) ListOptions(ctx context.Context, itemID string) ([]planning.Option, error) {
	response, err := g.client.ListBookingOptions(outgoingContext(ctx), &appointmentservice.ListBookingOptionsRequest{ItemId: itemID})
	if err != nil {
		return nil, mapPlanningError("list planning booking options", err)
	}
	result := make([]planning.Option, 0, len(response.GetOptions()))
	for _, value := range response.GetOptions() {
		result = append(result, planning.Option{ItemID: value.GetItemId(), RoomID: value.GetRoomId(), RoomDisplayName: value.GetRoomDisplayName(), CampusID: value.GetCampusId(), Building: value.GetBuilding(), FloorNumber: value.GetFloorNumber(), RoomNumber: value.GetRoomNumber(), ServiceDate: value.GetServiceDate(), Session: value.GetSession(), RemainingCapacity: value.GetRemainingCapacity(), EstimatedDurationMinutes: value.GetEstimatedDurationMinutes()})
	}
	return result, nil
}

func (g *Planning) ListMyBookings(ctx context.Context, view string) ([]planning.Booking, error) {
	response, err := g.client.ListMyBookings(outgoingContext(ctx), &appointmentservice.ListMyBookingsRequest{Page: 1, PageSize: 100, View: view})
	if err != nil {
		return nil, mapPlanningError("list planning bookings", err)
	}
	result := make([]planning.Booking, 0, len(response.GetBookings()))
	for _, value := range response.GetBookings() {
		result = append(result, planning.Booking{BookingID: value.GetBookingId(), ItemID: value.GetItemId(), ItemName: value.GetItemName(), RoomID: value.GetRoomId(), RoomDisplayName: value.GetRoomDisplayName(), CampusID: value.GetCampusId(), Building: value.GetBuilding(), FloorNumber: value.GetFloorNumber(), RoomNumber: value.GetRoomNumber(), EstimatedDurationMinutes: value.GetEstimatedDurationMinutes(), ServiceDate: value.GetServiceDate(), Session: value.GetSession(), Status: value.GetStatus()})
	}
	return result, nil
}

func (g *Planning) CreateBookingBatch(ctx context.Context, command planning.ConfirmCommand, items []planning.PlanItem) ([]string, error) {
	requestItems := make([]*appointmentservice.BookingBatchItem, 0, len(items))
	for _, value := range items {
		requestItems = append(requestItems, &appointmentservice.BookingBatchItem{ItemId: value.ItemID, RoomId: value.RoomID, ServiceDate: value.ServiceDate, Session: value.Session})
	}
	response, err := g.client.CreateBookingBatch(outgoingContext(ctx), &appointmentservice.CreateBookingBatchRequest{Items: requestItems, OperationId: command.OperationID, RequestId: command.RequestID, PatientDisplayName: command.PatientDisplayName, PatientPhoneMasked: command.PatientPhoneMasked})
	if err != nil {
		return nil, mapPlanningError("create smart booking batch", err)
	}
	ids := make([]string, 0, len(response.GetBookings()))
	for _, value := range response.GetBookings() {
		ids = append(ids, value.GetBookingId())
	}
	return ids, nil
}

func mapPlanningError(action string, err error) error {
	value := status.Convert(err)
	for _, detail := range value.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.GetReason() == "PATIENT_ITEM_SESSION_OCCUPIED" {
			return planning.ErrExistingBooking
		}
	}
	switch status.Code(err) {
	case codes.InvalidArgument:
		return planning.ErrInvalid
	case codes.Unauthenticated, codes.PermissionDenied:
		return planning.ErrForbidden
	case codes.NotFound:
		return planning.ErrNotFound
	case codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition, codes.ResourceExhausted:
		return planning.ErrConflict
	case codes.Unavailable, codes.DeadlineExceeded:
		return planning.ErrUnavailable
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

var _ planning.Appointment = (*Planning)(nil)
