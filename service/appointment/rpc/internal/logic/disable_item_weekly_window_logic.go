package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableItemWeeklyWindowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableItemWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableItemWeeklyWindowLogic {
	return &DisableItemWeeklyWindowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisableItemWeeklyWindowLogic) DisableItemWeeklyWindow(in *appointmentv1.ChangeResourceStatusRequest) (*appointmentv1.ItemWeeklyWindow, error) {
	return disableItemWindow(l.ctx, l.svcCtx, in)
}
