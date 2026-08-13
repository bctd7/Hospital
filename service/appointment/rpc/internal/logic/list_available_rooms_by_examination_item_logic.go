package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAvailableRoomsByExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAvailableRoomsByExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAvailableRoomsByExaminationItemLogic {
	return &ListAvailableRoomsByExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAvailableRoomsByExaminationItemLogic) ListAvailableRoomsByExaminationItem(in *appointmentv1.ListAvailableRoomsByExaminationItemRequest) (*appointmentv1.ListRoomExaminationItemsResponse, error) {
	return listItemRooms(l.ctx, l.svcCtx, in)
}
