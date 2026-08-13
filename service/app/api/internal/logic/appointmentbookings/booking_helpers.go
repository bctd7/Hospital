package appointmentbookings

import (
	"context"

	"hospital/common/observability/logging"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func bookingRPCContext(ctx context.Context, svcCtx *svc.ServiceContext) (context.Context, string, error) {
	rpcCtx, err := svcCtx.AuthenticatedRPCContext(ctx)
	return rpcCtx, logging.RequestIDFromContext(ctx), err
}

func booking(value *appointmentv1.Booking) *types.BookingResponse {
	return &types.BookingResponse{
		BookingID: value.BookingId, PatientAccountID: value.PatientAccountId,
		DepartmentID: value.DepartmentId, ItemID: value.ItemId, ItemName: value.ItemName,
		RoomID: value.RoomId, RoomDisplayName: value.RoomDisplayName, CampusID: value.CampusId,
		ServiceDate: value.ServiceDate, Session: value.Session, Status: value.Status,
		RoomOpenTime: value.RoomOpenTime, RoomCloseTime: value.RoomCloseTime,
		ItemStartTime: value.ItemStartTime, ItemEndTime: value.ItemEndTime,
		BookingCutoffTime: value.BookingCutoffTime, Version: value.Version,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
		CheckedInAt: value.CheckedInAt, CheckedInBy: value.CheckedInBy,
	}
}

func bookingList(value *appointmentv1.ListBookingsResponse) *types.ListBookingsAPIResponse {
	response := &types.ListBookingsAPIResponse{Page: value.Page, PageSize: value.PageSize, Total: value.Total}
	response.Bookings = make([]types.BookingResponse, 0, len(value.Bookings))
	for _, current := range value.Bookings {
		response.Bookings = append(response.Bookings, *booking(current))
	}
	return response
}
