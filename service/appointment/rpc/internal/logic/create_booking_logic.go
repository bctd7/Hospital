package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateBookingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBookingLogic {
	return &CreateBookingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateBookingLogic) CreateBooking(in *appointmentv1.CreateBookingRequest) (*appointmentv1.Booking, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.PatientManager.CreateBooking(l.ctx, principal, patientmanager.CreateBookingCommand{
		ItemID: in.GetItemId(), RoomID: in.GetRoomId(), ServiceDate: in.GetServiceDate(),
		Session:            patientmanager.Session(in.GetSession()),
		PatientDisplayName: in.GetPatientDisplayName(), PatientPhoneMasked: in.GetPatientPhoneMasked(),
		OperationID: in.GetOperationId(), RequestID: in.GetRequestId(),
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingResponse(value), nil
}
