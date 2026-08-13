package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRoomExaminationItemsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListRoomExaminationItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRoomExaminationItemsLogic {
	return &ListRoomExaminationItemsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListRoomExaminationItemsLogic) ListRoomExaminationItems(in *appointmentv1.ListRoomExaminationItemsRequest) (*appointmentv1.ListRoomExaminationItemsResponse, error) {
	return listRoomItems(l.ctx, l.svcCtx, in)
}
