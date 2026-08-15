package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkMessageReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkMessageReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkMessageReadLogic {
	return &MarkMessageReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *MarkMessageReadLogic) MarkMessageRead(in *appointmentv1.MarkMessageReadRequest) (*appointmentv1.Message, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.StaffManager.MarkMessageRead(l.ctx, principal, in.GetDepartmentId(), in.GetMessageKey())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return messageResponse(value), nil
}
