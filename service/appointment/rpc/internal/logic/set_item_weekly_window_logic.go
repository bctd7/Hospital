package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetItemWeeklyWindowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetItemWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetItemWeeklyWindowLogic {
	return &SetItemWeeklyWindowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetItemWeeklyWindowLogic) SetItemWeeklyWindow(in *appointmentv1.SetItemWeeklyWindowRequest) (*appointmentv1.ItemWeeklyWindow, error) {
	return setItemWindow(l.ctx, l.svcCtx, in)
}
