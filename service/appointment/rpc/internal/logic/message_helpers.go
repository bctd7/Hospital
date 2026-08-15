package logic

import (
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
)

func messageResponse(value common.Message) *appointmentv1.Message {
	response := &appointmentv1.Message{
		MessageKey: value.MessageKey, MessageType: string(value.MessageType),
		OccurredAt: value.OccurredAt.UTC().Format(timeLayout), Booking: bookingResponse(value.Booking),
		ReportId: value.ReportID, ReportVersionId: value.ReportVersionID, ReportVersionNo: value.ReportVersionNo,
	}
	if value.ReadAt != nil {
		response.ReadAt = value.ReadAt.UTC().Format(timeLayout)
	}
	return response
}

func messageListResponse(page common.MessagePage) *appointmentv1.ListMessagesResponse {
	response := &appointmentv1.ListMessagesResponse{Page: page.Page, PageSize: page.PageSize, Total: page.Total, UnreadCount: page.UnreadCount}
	response.Messages = make([]*appointmentv1.Message, 0, len(page.Items))
	for _, value := range page.Items {
		response.Messages = append(response.Messages, messageResponse(value))
	}
	response.DepartmentUnreadCounts = make([]*appointmentv1.DepartmentUnreadCount, 0, len(page.DepartmentUnreadCounts))
	for _, value := range page.DepartmentUnreadCounts {
		response.DepartmentUnreadCounts = append(response.DepartmentUnreadCounts, &appointmentv1.DepartmentUnreadCount{DepartmentId: value.DepartmentID, UnreadCount: value.UnreadCount})
	}
	return response
}
