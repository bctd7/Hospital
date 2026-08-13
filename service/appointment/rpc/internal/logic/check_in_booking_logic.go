package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckInBookingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckInBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckInBookingLogic {
	return &CheckInBookingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CheckInBookingLogic) CheckInBooking(in *appointmentv1.CheckInBookingRequest) (*appointmentv1.Booking, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.CheckInBooking(l.ctx, principal, staffmanager.CheckInBookingCommand{
		BookingID: in.GetBookingId(), ExpectedVersion: in.GetExpectedVersion(),
		OperationID: in.GetOperationId(), RequestID: in.GetRequestId(),
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingResponse(value), nil
}
