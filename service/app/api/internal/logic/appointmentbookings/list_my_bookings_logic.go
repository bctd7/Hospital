// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentbookings

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyBookingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMyBookingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyBookingsLogic {
	return &ListMyBookingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMyBookingsLogic) ListMyBookings(req *types.ListMyBookingsAPIRequest) (resp *types.ListBookingsAPIResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.ListMyBookings(rpcCtx, &appointmentv1.ListMyBookingsRequest{Page: req.Page, PageSize: req.PageSize, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return bookingList(value), nil
}
