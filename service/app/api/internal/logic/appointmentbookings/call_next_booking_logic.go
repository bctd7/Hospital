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

type CallNextBookingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCallNextBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CallNextBookingLogic {
	return &CallNextBookingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CallNextBookingLogic) CallNextBooking(req *types.CallNextBookingAPIRequest) (resp *types.BookingResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.CallNextBooking(rpcCtx, &appointmentv1.CallNextBookingRequest{
		DepartmentId: req.DepartmentID, RoomId: req.RoomID, ServiceDate: req.ServiceDate, OperationId: req.OperationID, RequestId: requestID,
	})
	if err != nil {
		return nil, err
	}
	return bookingWithOrganization(rpcCtx, l.svcCtx, requestID, value), nil
}
