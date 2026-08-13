package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableRoomWeeklyWindowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableRoomWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableRoomWeeklyWindowLogic {
	return &DisableRoomWeeklyWindowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisableRoomWeeklyWindowLogic) DisableRoomWeeklyWindow(in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.RoomWeeklyWindow, error) {
	return disableRoomWindow(l.ctx, l.svcCtx, in)
}
