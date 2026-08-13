package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRoomWeeklyWindowsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListRoomWeeklyWindowsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRoomWeeklyWindowsLogic {
	return &ListRoomWeeklyWindowsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListRoomWeeklyWindowsLogic) ListRoomWeeklyWindows(in *appointmentv1.ListWeeklyWindowsRequest) (*appointmentv1.ListRoomWeeklyWindowsResponse, error) {
	return listRoomWindows(l.ctx, l.svcCtx, in)
}
