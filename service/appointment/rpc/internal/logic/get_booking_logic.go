package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetBookingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBookingLogic {
	return &GetBookingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetBookingLogic) GetBooking(in *appointmentv1.GetBookingRequest) (*appointmentv1.Booking, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.GetBooking(l.ctx, principal, in.GetBookingId())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingResponse(value), nil
}
