// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableRoomExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableRoomExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableRoomExaminationItemLogic {
	return &DisableRoomExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableRoomExaminationItemLogic) DisableRoomExaminationItem(req *types.ChangeAppointmentResourceStatusRequest) (resp *types.RoomExaminationItemResponse, err error) {
	return changeRoomItem(l.ctx, l.svcCtx, req, false)
}
