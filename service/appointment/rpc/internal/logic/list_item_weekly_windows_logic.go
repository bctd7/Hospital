package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListItemWeeklyWindowsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListItemWeeklyWindowsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListItemWeeklyWindowsLogic {
	return &ListItemWeeklyWindowsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListItemWeeklyWindowsLogic) ListItemWeeklyWindows(in *appointmentv1.ListWeeklyWindowsRequest) (*appointmentv1.ListItemWeeklyWindowsResponse, error) {
	return listItemWindows(l.ctx, l.svcCtx, in)
}
