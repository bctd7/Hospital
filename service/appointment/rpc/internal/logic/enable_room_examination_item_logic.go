package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnableRoomExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnableRoomExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableRoomExaminationItemLogic {
	return &EnableRoomExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EnableRoomExaminationItemLogic) EnableRoomExaminationItem(in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.RoomExaminationItem, error) {
	return changeRoomItem(l.ctx, l.svcCtx, in, true)
}
