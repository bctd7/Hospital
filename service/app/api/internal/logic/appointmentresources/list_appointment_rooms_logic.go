// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppointmentRoomsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAppointmentRoomsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppointmentRoomsLogic {
	return &ListAppointmentRoomsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppointmentRoomsLogic) ListAppointmentRooms(req *types.ListAppointmentRoomsRequest) (resp *types.ListAppointmentRoomsResponse, err error) {
	return listRooms(l.ctx, l.svcCtx, req)
}
