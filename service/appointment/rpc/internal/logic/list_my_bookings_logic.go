package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyBookingsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMyBookingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyBookingsLogic {
	return &ListMyBookingsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListMyBookingsLogic) ListMyBookings(in *appointmentv1.ListMyBookingsRequest) (*appointmentv1.ListBookingsResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	page, err := l.svcCtx.PatientManager.ListMyBookings(l.ctx, principal, patientmanager.BookingListView(in.GetView()), in.GetPage(), in.GetPageSize())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingListResponse(page.Items, page.Page, page.PageSize, page.Total), nil
}
