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

type CheckInBookingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckInBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckInBookingLogic {
	return &CheckInBookingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CheckInBookingLogic) CheckInBooking(req *types.CheckInBookingAPIRequest) (resp *types.BookingResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.CheckInBooking(rpcCtx, &appointmentv1.CheckInBookingRequest{BookingId: req.BookingID, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return booking(value), nil
}
