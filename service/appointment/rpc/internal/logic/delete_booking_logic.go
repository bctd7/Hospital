package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteBookingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBookingLogic {
	return &DeleteBookingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteBookingLogic) DeleteBooking(in *appointmentv1.DeleteBookingRequest) (*appointmentv1.DeleteBookingResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	bookingID, err := l.svcCtx.StaffManager.DeleteBooking(l.ctx, principal, staffmanager.DeleteBookingCommand{
		BookingID: in.GetBookingId(), OperationID: in.GetOperationId(), Reason: in.GetReason(), RequestID: in.GetRequestId(),
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return &appointmentv1.DeleteBookingResponse{BookingId: bookingID, Deleted: true}, nil
}
