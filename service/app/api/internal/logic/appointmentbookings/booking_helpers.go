package appointmentbookings

import (
	"context"
	"strings"

	"hospital/common/observability/logging"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type displaySnapshots struct {
	PatientName string
	MaskedPhone string
	ActorName   string
}

func currentDisplaySnapshots(ctx context.Context, svcCtx *svc.ServiceContext, requestID string) (displaySnapshots, error) {
	value, err := svcCtx.Identity.GetAccountDisplayProfile(ctx, &identityv1.GetAccountDisplayProfileRequest{RequestId: requestID})
	if err != nil {
		return displaySnapshots{}, err
	}
	nickname := strings.TrimSpace(value.GetNickname())
	maskedPhone := strings.TrimSpace(value.GetMaskedPhone())
	patientName := nickname
	if patientName == "" {
		patientName = "患者"
	}
	actorName := strings.TrimSpace(value.GetStaffDisplayName())
	if actorName == "" {
		actorName = nickname
	}
	if actorName == "" {
		actorName = "工作人员"
	}
	return displaySnapshots{PatientName: patientName, MaskedPhone: maskedPhone, ActorName: actorName}, nil
}

func reportOrganizationSnapshots(ctx context.Context, svcCtx *svc.ServiceContext, booking *appointmentv1.Booking, requestID string) (string, string, error) {
	value, err := svcCtx.Identity.ListDepartments(ctx, &identityv1.ListDepartmentsRequest{CampusId: booking.GetCampusId(), RequestId: requestID})
	if err != nil {
		return "", "", err
	}
	for _, department := range value.GetItems() {
		if department.GetDepartmentId() == booking.GetDepartmentId() {
			return department.GetName(), department.GetCampusName(), nil
		}
	}
	return "", "", identitySnapshotNotFound()
}

func bookingRPCContext(ctx context.Context, svcCtx *svc.ServiceContext) (context.Context, string, error) {
	rpcCtx, err := svcCtx.AuthenticatedRPCContext(ctx)
	return rpcCtx, logging.RequestIDFromContext(ctx), err
}

func booking(value *appointmentv1.Booking) *types.BookingResponse {
	return &types.BookingResponse{
		BookingID: value.BookingId, PatientAccountID: value.PatientAccountId,
		PatientDisplayName: value.PatientDisplayName, PatientPhoneMasked: value.PatientPhoneMasked,
		DepartmentID: value.DepartmentId, ItemID: value.ItemId, ItemName: value.ItemName,
		RoomID: value.RoomId, RoomDisplayName: value.RoomDisplayName, CampusID: value.CampusId,
		Building: value.Building, FloorNumber: value.FloorNumber, RoomNumber: value.RoomNumber,
		ServiceDate: value.ServiceDate, Session: value.Session, Status: value.Status,
		RoomOpenTime: value.RoomOpenTime, RoomCloseTime: value.RoomCloseTime,
		ItemStartTime: value.ItemStartTime, ItemEndTime: value.ItemEndTime,
		BookingCutoffTime: value.BookingCutoffTime, Version: value.Version,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
		StartedAt: value.StartedAt, StartedBy: value.StartedBy,
		StartedByDisplayName: value.StartedByDisplayName,
		CompletedAt:          value.CompletedAt, CompletedBy: value.CompletedBy,
		CompletedByDisplayName: value.CompletedByDisplayName,
	}
}

// bookingWithOrganization 为读取页面补齐可读科室和院区名称；目录暂不可用时仍返回预约本身。
func bookingWithOrganization(ctx context.Context, svcCtx *svc.ServiceContext, requestID string, value *appointmentv1.Booking) *types.BookingResponse {
	response := booking(value)
	enrichBookingOrganizations(ctx, svcCtx, requestID, []*appointmentv1.Booking{value}, []*types.BookingResponse{response})
	return response
}

func bookingList(value *appointmentv1.ListBookingsResponse) *types.ListBookingsAPIResponse {
	response := &types.ListBookingsAPIResponse{Page: value.Page, PageSize: value.PageSize, Total: value.Total}
	response.Bookings = make([]types.BookingResponse, 0, len(value.Bookings))
	for _, current := range value.Bookings {
		response.Bookings = append(response.Bookings, *booking(current))
	}
	return response
}

// bookingListWithOrganizations 按院区去重读取 Identity 目录，避免列表中每条预约分别发起 RPC。
func bookingListWithOrganizations(ctx context.Context, svcCtx *svc.ServiceContext, requestID string, value *appointmentv1.ListBookingsResponse) *types.ListBookingsAPIResponse {
	response := bookingList(value)
	items := make([]*types.BookingResponse, 0, len(response.Bookings))
	for index := range response.Bookings {
		items = append(items, &response.Bookings[index])
	}
	enrichBookingOrganizations(ctx, svcCtx, requestID, value.GetBookings(), items)
	return response
}

func enrichBookingOrganizations(ctx context.Context, svcCtx *svc.ServiceContext, requestID string, bookings []*appointmentv1.Booking, responses []*types.BookingResponse) {
	departmentsByCampus := make(map[string]map[string]*identityv1.DepartmentSummary)
	for index, value := range bookings {
		if value == nil || index >= len(responses) || responses[index] == nil {
			continue
		}
		campusID := value.GetCampusId()
		departments, loaded := departmentsByCampus[campusID]
		if !loaded {
			departments = make(map[string]*identityv1.DepartmentSummary)
			directory, err := svcCtx.Identity.ListDepartments(ctx, &identityv1.ListDepartmentsRequest{CampusId: campusID, RequestId: requestID})
			if err == nil {
				for _, department := range directory.GetItems() {
					departments[department.GetDepartmentId()] = department
				}
			}
			departmentsByCampus[campusID] = departments
		}
		if department := departments[value.GetDepartmentId()]; department != nil {
			responses[index].DepartmentName = department.GetName()
			responses[index].CampusName = department.GetCampusName()
		}
	}
}
