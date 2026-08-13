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

type DeleteBookingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBookingLogic {
	return &DeleteBookingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteBookingLogic) DeleteBooking(req *types.DeleteBookingAPIRequest) (resp *types.DeleteBookingAPIResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.DeleteBooking(rpcCtx, &appointmentv1.DeleteBookingRequest{BookingId: req.BookingID, OperationId: req.OperationID, Reason: req.Reason, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return &types.DeleteBookingAPIResponse{BookingID: value.BookingId, Deleted: value.Deleted}, nil
}
