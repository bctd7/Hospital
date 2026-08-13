package logic

import (
	"errors"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
)

func bookingRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, common.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid booking request")
	case errors.Is(err, common.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, common.ErrNotFound):
		return status.Error(codes.NotFound, "booking or booking resource not found")
	case errors.Is(err, common.ErrCapacityFull):
		return status.Error(codes.ResourceExhausted, "booking capacity is full")
	case errors.Is(err, common.ErrPatientSessionOccupied):
		return bookingStatusError(codes.FailedPrecondition, "patient already has a pending booking in this date and session", "PATIENT_SESSION_OCCUPIED")
	case errors.Is(err, common.ErrBookingClosed), errors.Is(err, common.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "booking state does not allow the operation")
	case errors.Is(err, common.ErrConflict), errors.Is(err, common.ErrVersionConflict):
		return status.Error(codes.Aborted, "booking data conflict")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func bookingStatusError(code codes.Code, message, reason string) error {
	value := status.New(code, message)
	withDetails, err := value.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: "hospital.appointment",
	})
	if err != nil {
		return value.Err()
	}
	return withDetails.Err()
}

func bookingResponse(value common.Booking) *appointmentv1.Booking {
	response := &appointmentv1.Booking{
		BookingId: value.BookingID, PatientAccountId: value.PatientAccountID,
		DepartmentId: value.DepartmentID, ItemId: value.ItemID, ItemName: value.ItemName,
		RoomId: value.RoomID, RoomDisplayName: value.RoomDisplayName, CampusId: value.CampusID,
		ServiceDate: value.ServiceDate.Format("2006-01-02"), Session: string(value.Session),
		Status: string(value.Status), RoomOpenTime: value.RoomOpenTime, RoomCloseTime: value.RoomCloseTime,
		ItemStartTime: value.ItemStartTime, ItemEndTime: value.ItemEndTime,
		BookingCutoffTime: value.BookingCutoffTime, Version: value.Version,
		CreatedAt: value.CreatedAt.UTC().Format(timeLayout), UpdatedAt: value.UpdatedAt.UTC().Format(timeLayout),
		CheckedInBy: value.CheckedInBy,
	}
	if value.CheckedInAt != nil {
		response.CheckedInAt = value.CheckedInAt.UTC().Format(timeLayout)
	}
	return response
}

func bookingListResponse(values []common.Booking, page, pageSize, total int64) *appointmentv1.ListBookingsResponse {
	response := &appointmentv1.ListBookingsResponse{Page: page, PageSize: pageSize, Total: total}
	response.Bookings = make([]*appointmentv1.Booking, 0, len(values))
	for _, value := range values {
		response.Bookings = append(response.Bookings, bookingResponse(value))
	}
	return response
}
