package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type MarkMyMessageReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMarkMyMessageReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkMyMessageReadLogic {
	return &MarkMyMessageReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *MarkMyMessageReadLogic) MarkMyMessageRead(in *appointmentv1.MarkMessageReadRequest) (*appointmentv1.Message, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.PatientManager.MarkMyMessageRead(l.ctx, principal, in.GetMessageKey())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return messageResponse(value), nil
}
