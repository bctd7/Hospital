// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppointmentRoomLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAppointmentRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppointmentRoomLogic {
	return &UpdateAppointmentRoomLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAppointmentRoomLogic) UpdateAppointmentRoom(req *types.UpdateAppointmentRoomRequest) (resp *types.AppointmentRoomResponse, err error) {
	return updateRoom(l.ctx, l.svcCtx, req)
}
