package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RetireRoomLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRetireRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RetireRoomLogic {
	return &RetireRoomLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RetireRoomLogic) RetireRoom(in *appointmentv1.RetireRoomRequest) (*appointmentv1.Room, error) {
	return retireRoom(l.ctx, l.svcCtx, in)
}
