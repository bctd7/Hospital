package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddRoomExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddRoomExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddRoomExaminationItemLogic {
	return &AddRoomExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AddRoomExaminationItemLogic) AddRoomExaminationItem(in *appointmentv1.AddRoomExaminationItemRequest) (*appointmentv1.RoomExaminationItem, error) {
	return addRoomItem(l.ctx, l.svcCtx, in)
}
