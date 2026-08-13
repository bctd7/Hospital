package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnableRoomLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnableRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableRoomLogic {
	return &EnableRoomLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EnableRoomLogic) EnableRoom(in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.Room, error) {
	return changeRoom(l.ctx, l.svcCtx, in, true)
}
