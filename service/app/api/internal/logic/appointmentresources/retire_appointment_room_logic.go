// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RetireAppointmentRoomLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRetireAppointmentRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RetireAppointmentRoomLogic {
	return &RetireAppointmentRoomLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RetireAppointmentRoomLogic) RetireAppointmentRoom(req *types.ChangeAppointmentResourceStatusRequest) (resp *types.AppointmentRoomResponse, err error) {
	return retireRoom(l.ctx, l.svcCtx, req)
}
