package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartExaminationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartExaminationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartExaminationLogic {
	return &StartExaminationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StartExaminationLogic) StartExamination(in *appointmentv1.StartExaminationRequest) (*appointmentv1.Booking, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.StartExamination(l.ctx, principal, staffinput.StartExamination{
		BookingID: in.GetBookingId(), ExpectedVersion: in.GetExpectedVersion(), ActorDisplayName: in.GetActorDisplayName(),
		Operation: staffinput.Operation{OperationID: in.GetOperationId(), RequestID: in.GetRequestId()},
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return bookingResponse(value), nil
}
