// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableRoomWeeklyWindowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableRoomWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableRoomWeeklyWindowLogic {
	return &DisableRoomWeeklyWindowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableRoomWeeklyWindowLogic) DisableRoomWeeklyWindow(req *types.ChangeAppointmentResourceStatusRequest) (resp *types.RoomWeeklyWindowResponse, err error) {
	return disableRoomWindow(l.ctx, l.svcCtx, req)
}
