package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyBookingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyBookingLogic {
	return &GetMyBookingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMyBookingLogic) GetMyBooking(in *appointmentv1.GetBookingRequest) (*appointmentv1.Booking, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.PatientManager.GetMyBooking(l.ctx, principal, in.GetBookingId())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingResponse(value), nil
}
