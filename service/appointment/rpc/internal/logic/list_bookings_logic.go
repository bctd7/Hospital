package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListBookingsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListBookingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBookingsLogic {
	return &ListBookingsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListBookingsLogic) ListBookings(in *appointmentv1.ListBookingsRequest) (*appointmentv1.ListBookingsResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	page, err := l.svcCtx.StaffManager.ListBookings(l.ctx, principal, staffmanager.ListBookingsQuery{
		DepartmentID: in.GetDepartmentId(), ServiceDate: in.GetServiceDate(),
		Session: staffmanager.Session(in.GetSession()), ItemID: in.GetItemId(), RoomID: in.GetRoomId(),
		Status: staffmanager.BookingStatus(in.GetStatus()), Page: in.GetPage(), PageSize: in.GetPageSize(),
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingListResponse(page.Items, page.Page, page.PageSize, page.Total), nil
}
