package appointmentbookings

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

func message(value *appointmentv1.Message) *types.MessageResponse {
	return &types.MessageResponse{
		MessageKey: value.GetMessageKey(), MessageType: value.GetMessageType(), OccurredAt: value.GetOccurredAt(), ReadAt: value.GetReadAt(),
		Booking: *booking(value.GetBooking()), ReportID: value.GetReportId(), ReportVersionID: value.GetReportVersionId(), ReportVersionNo: value.GetReportVersionNo(),
	}
}

func messageListWithOrganizations(ctx context.Context, svcCtx *svc.ServiceContext, requestID string, value *appointmentv1.ListMessagesResponse) *types.ListMessagesAPIResponse {
	response := &types.ListMessagesAPIResponse{Page: value.GetPage(), PageSize: value.GetPageSize(), Total: value.GetTotal(), UnreadCount: value.GetUnreadCount()}
	response.Messages = make([]types.MessageResponse, 0, len(value.GetMessages()))
	bookings := make([]*appointmentv1.Booking, 0, len(value.GetMessages()))
	bookingResponses := make([]*types.BookingResponse, 0, len(value.GetMessages()))
	for _, current := range value.GetMessages() {
		item := message(current)
		response.Messages = append(response.Messages, *item)
		bookings = append(bookings, current.GetBooking())
		bookingResponses = append(bookingResponses, &response.Messages[len(response.Messages)-1].Booking)
	}
	enrichBookingOrganizations(ctx, svcCtx, requestID, bookings, bookingResponses)
	response.DepartmentUnreadCounts = make([]types.DepartmentUnreadCountResponse, 0, len(value.GetDepartmentUnreadCounts()))
	for _, current := range value.GetDepartmentUnreadCounts() {
		response.DepartmentUnreadCounts = append(response.DepartmentUnreadCounts, types.DepartmentUnreadCountResponse{DepartmentID: current.GetDepartmentId(), UnreadCount: current.GetUnreadCount()})
	}
	return response
}

func messageWithOrganization(ctx context.Context, svcCtx *svc.ServiceContext, requestID string, value *appointmentv1.Message) *types.MessageResponse {
	response := message(value)
	enrichBookingOrganizations(ctx, svcCtx, requestID, []*appointmentv1.Booking{value.GetBooking()}, []*types.BookingResponse{&response.Booking})
	return response
}
