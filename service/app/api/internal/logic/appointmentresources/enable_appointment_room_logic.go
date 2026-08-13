// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnableAppointmentRoomLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnableAppointmentRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableAppointmentRoomLogic {
	return &EnableAppointmentRoomLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *EnableAppointmentRoomLogic) EnableAppointmentRoom(req *types.ChangeAppointmentResourceStatusRequest) (resp *types.AppointmentRoomResponse, err error) {
	return changeRoom(l.ctx, l.svcCtx, req, true)
}
