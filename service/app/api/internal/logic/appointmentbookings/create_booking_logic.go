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

type CreateBookingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBookingLogic {
	return &CreateBookingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateBookingLogic) CreateBooking(req *types.CreateBookingAPIRequest) (resp *types.BookingResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	display, err := currentDisplaySnapshots(rpcCtx, l.svcCtx, requestID)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.CreateBooking(rpcCtx, &appointmentv1.CreateBookingRequest{
		ItemId: req.ItemID, RoomId: req.RoomID, ServiceDate: req.ServiceDate,
		Session: req.Session, OperationId: req.OperationID, RequestId: requestID,
		PatientDisplayName: display.PatientName, PatientPhoneMasked: display.MaskedPhone,
	})
	if err != nil {
		return nil, err
	}
	return bookingWithOrganization(rpcCtx, l.svcCtx, requestID, value), nil
}
