// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableAppointmentRoomLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableAppointmentRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableAppointmentRoomLogic {
	return &DisableAppointmentRoomLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableAppointmentRoomLogic) DisableAppointmentRoom(req *types.ChangeAppointmentResourceStatusRequest) (resp *types.AppointmentRoomResponse, err error) {
	return changeRoom(l.ctx, l.svcCtx, req, false)
}
