package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteMyBookingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteMyBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMyBookingLogic {
	return &DeleteMyBookingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeleteMyBookingLogic) DeleteMyBooking(in *appointmentv1.DeleteBookingRequest) (*appointmentv1.DeleteBookingResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	bookingID, err := l.svcCtx.PatientManager.DeleteMyBooking(l.ctx, principal, patientmanager.DeleteBookingCommand{
		BookingID: in.GetBookingId(), OperationID: in.GetOperationId(), Reason: in.GetReason(), RequestID: in.GetRequestId(),
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return &appointmentv1.DeleteBookingResponse{BookingId: bookingID, Deleted: true}, nil
}
