// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddRoomExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddRoomExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddRoomExaminationItemLogic {
	return &AddRoomExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddRoomExaminationItemLogic) AddRoomExaminationItem(req *types.AddRoomExaminationItemAPIRequest) (resp *types.RoomExaminationItemResponse, err error) {
	return addRoomItem(l.ctx, l.svcCtx, req)
}
