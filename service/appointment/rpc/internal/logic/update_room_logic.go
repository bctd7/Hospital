package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateRoomLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoomLogic {
	return &UpdateRoomLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateRoomLogic) UpdateRoom(in *appointmentv1.UpdateRoomRequest) (*appointmentv1.Room, error) {
	return updateRoom(l.ctx, l.svcCtx, in)
}
