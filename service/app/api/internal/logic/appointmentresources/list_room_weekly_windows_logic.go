// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRoomWeeklyWindowsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListRoomWeeklyWindowsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRoomWeeklyWindowsLogic {
	return &ListRoomWeeklyWindowsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListRoomWeeklyWindowsLogic) ListRoomWeeklyWindows(req *types.WeeklyWindowsPathRequest) (resp *types.ListRoomWeeklyWindowsAPIResponse, err error) {
	return listRoomWindows(l.ctx, l.svcCtx, req)
}
