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

type MarkMyMessageReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkMyMessageReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkMyMessageReadLogic {
	return &MarkMyMessageReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkMyMessageReadLogic) MarkMyMessageRead(req *types.MarkMyMessageReadAPIRequest) (resp *types.MessageResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.MarkMyMessageRead(rpcCtx, &appointmentv1.MarkMessageReadRequest{MessageKey: req.MessageKey, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return messageWithOrganization(rpcCtx, l.svcCtx, requestID, value), nil
}
