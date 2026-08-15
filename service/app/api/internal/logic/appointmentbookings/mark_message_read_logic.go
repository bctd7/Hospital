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

type MarkMessageReadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMarkMessageReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MarkMessageReadLogic {
	return &MarkMessageReadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MarkMessageReadLogic) MarkMessageRead(req *types.MarkMessageReadAPIRequest) (resp *types.MessageResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.MarkMessageRead(rpcCtx, &appointmentv1.MarkMessageReadRequest{DepartmentId: req.DepartmentID, MessageKey: req.MessageKey, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return messageWithOrganization(rpcCtx, l.svcCtx, requestID, value), nil
}
