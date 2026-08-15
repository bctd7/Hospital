package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CallNextBookingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCallNextBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CallNextBookingLogic {
	return &CallNextBookingLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CallNextBookingLogic) CallNextBooking(in *appointmentv1.CallNextBookingRequest) (*appointmentv1.Booking, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.CallNext(l.ctx, principal, staffinput.CallNext{
		DepartmentID: in.GetDepartmentId(), RoomID: in.GetRoomId(), ServiceDate: in.GetServiceDate(),
		OperationID: in.GetOperationId(), RequestID: in.GetRequestId(),
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingResponse(value), nil
}
