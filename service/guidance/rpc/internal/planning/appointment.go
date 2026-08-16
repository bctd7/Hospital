package planning

import (
	"context"
	"fmt"

	"hospital/service/appointment/rpc/appointmentservice"
	"hospital/service/guidance/rpc/internal/projectconfiguration"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AppointmentGateway struct {
	client appointmentservice.AppointmentService
}

func NewAppointmentGateway(client appointmentservice.AppointmentService) (*AppointmentGateway, error) {
	if client == nil {
		return nil, fmt.Errorf("appointment client is required")
	}
	return &AppointmentGateway{client: client}, nil
}

func planningContext(ctx context.Context) context.Context {
	if incoming, ok := metadata.FromIncomingContext(ctx); ok {
		return metadata.NewOutgoingContext(ctx, incoming.Copy())
	}
	return ctx
}

func (g *AppointmentGateway) ResolveProject(ctx context.Context, itemID string) (projectconfiguration.Project, error) {
	value, err := g.client.GetExaminationItemReference(planningContext(ctx), &appointmentservice.GetExaminationItemRequest{ItemId: itemID})
	if err != nil {
		return projectconfiguration.Project{}, fmt.Errorf("resolve planning project: %w", err)
	}
	return projectconfiguration.Project{ItemID: value.GetItemId(), OwnerDepartmentID: value.GetOwnerDepartmentId(), Name: value.GetName(), Status: value.GetStatus(), Version: value.GetVersion(), EstimatedDurationMinutes: value.GetEstimatedDurationMinutes()}, nil
}

func (g *AppointmentGateway) ListOptions(ctx context.Context, itemID string) ([]Option, error) {
	response, err := g.client.ListBookingOptions(planningContext(ctx), &appointmentservice.ListBookingOptionsRequest{ItemId: itemID})
	if err != nil {
		return nil, mapPlanningAppointmentError("list planning booking options", err)
	}
	result := make([]Option, 0, len(response.GetOptions()))
	for _, value := range response.GetOptions() {
		result = append(result, Option{ItemID: value.GetItemId(), RoomID: value.GetRoomId(), RoomDisplayName: value.GetRoomDisplayName(), CampusID: value.GetCampusId(), Building: value.GetBuilding(), FloorNumber: value.GetFloorNumber(), RoomNumber: value.GetRoomNumber(), ServiceDate: value.GetServiceDate(), Session: value.GetSession(), RemainingCapacity: value.GetRemainingCapacity(), EstimatedDurationMinutes: value.GetEstimatedDurationMinutes()})
	}
	return result, nil
}

func (g *AppointmentGateway) ListMyBookings(ctx context.Context, view string) ([]Booking, error) {
	response, err := g.client.ListMyBookings(planningContext(ctx), &appointmentservice.ListMyBookingsRequest{Page: 1, PageSize: 100, View: view})
	if err != nil {
		return nil, mapPlanningAppointmentError("list planning bookings", err)
	}
	result := make([]Booking, 0, len(response.GetBookings()))
	for _, value := range response.GetBookings() {
		result = append(result, Booking{BookingID: value.GetBookingId(), ItemID: value.GetItemId(), ItemName: value.GetItemName(), RoomID: value.GetRoomId(), RoomDisplayName: value.GetRoomDisplayName(), CampusID: value.GetCampusId(), Building: value.GetBuilding(), FloorNumber: value.GetFloorNumber(), RoomNumber: value.GetRoomNumber(), EstimatedDurationMinutes: value.GetEstimatedDurationMinutes(), ServiceDate: value.GetServiceDate(), Session: value.GetSession(), Status: value.GetStatus()})
	}
	return result, nil
}

func (g *AppointmentGateway) CreateBookingBatch(ctx context.Context, command ConfirmCommand, items []PlanItem) ([]string, error) {
	requestItems := make([]*appointmentservice.BookingBatchItem, 0, len(items))
	for _, value := range items {
		requestItems = append(requestItems, &appointmentservice.BookingBatchItem{ItemId: value.ItemID, RoomId: value.RoomID, ServiceDate: value.ServiceDate, Session: value.Session})
	}
	response, err := g.client.CreateBookingBatch(planningContext(ctx), &appointmentservice.CreateBookingBatchRequest{Items: requestItems, OperationId: command.OperationID, RequestId: command.RequestID, PatientDisplayName: command.PatientDisplayName, PatientPhoneMasked: command.PatientPhoneMasked})
	if err != nil {
		return nil, mapPlanningAppointmentError("create smart booking batch", err)
	}
	ids := make([]string, 0, len(response.GetBookings()))
	for _, value := range response.GetBookings() {
		ids = append(ids, value.GetBookingId())
	}
	return ids, nil
}

func mapPlanningAppointmentError(action string, err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return ErrInvalid
	case codes.Unauthenticated, codes.PermissionDenied:
		return ErrForbidden
	case codes.NotFound:
		return ErrNotFound
	case codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition, codes.ResourceExhausted:
		return ErrConflict
	case codes.Unavailable, codes.DeadlineExceeded:
		return ErrUnavailable
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

var _ Appointment = (*AppointmentGateway)(nil)
