// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAppointmentRoomLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateAppointmentRoomLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAppointmentRoomLogic {
	return &CreateAppointmentRoomLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateAppointmentRoomLogic) CreateAppointmentRoom(req *types.CreateAppointmentRoomRequest) (resp *types.AppointmentRoomResponse, err error) {
	return createRoom(l.ctx, l.svcCtx, req)
}
