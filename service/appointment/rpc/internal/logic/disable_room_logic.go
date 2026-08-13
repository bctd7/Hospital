package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableRoomLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableRoomLogic {
	return &DisableRoomLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisableRoomLogic) DisableRoom(in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.Room, error) {
	return changeRoom(l.ctx, l.svcCtx, in, false)
}
