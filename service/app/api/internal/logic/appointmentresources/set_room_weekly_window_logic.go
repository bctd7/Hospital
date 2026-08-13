// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetRoomWeeklyWindowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetRoomWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetRoomWeeklyWindowLogic {
	return &SetRoomWeeklyWindowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetRoomWeeklyWindowLogic) SetRoomWeeklyWindow(req *types.SetRoomWeeklyWindowAPIRequest) (resp *types.RoomWeeklyWindowResponse, err error) {
	return setRoomWindow(l.ctx, l.svcCtx, req)
}
