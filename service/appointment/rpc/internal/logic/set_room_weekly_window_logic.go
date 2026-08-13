package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetRoomWeeklyWindowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetRoomWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetRoomWeeklyWindowLogic {
	return &SetRoomWeeklyWindowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetRoomWeeklyWindowLogic) SetRoomWeeklyWindow(in *appointmentv1.SetRoomWeeklyWindowRequest) (*appointmentv1.RoomWeeklyWindow, error) {
	return setRoomWindow(l.ctx, l.svcCtx, in)
}
