package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableRoomExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableRoomExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableRoomExaminationItemLogic {
	return &DisableRoomExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisableRoomExaminationItemLogic) DisableRoomExaminationItem(in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.RoomExaminationItem, error) {
	return changeRoomItem(l.ctx, l.svcCtx, in, false)
}
