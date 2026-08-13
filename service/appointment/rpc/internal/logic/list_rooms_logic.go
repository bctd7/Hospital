package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRoomsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListRoomsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRoomsLogic {
	return &ListRoomsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListRoomsLogic) ListRooms(in *appointmentv1.ListRoomsRequest) (*appointmentv1.ListRoomsResponse, error) {
	return listRooms(l.ctx, l.svcCtx, in)
}
