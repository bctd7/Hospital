// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetItemWeeklyWindowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetItemWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetItemWeeklyWindowLogic {
	return &SetItemWeeklyWindowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetItemWeeklyWindowLogic) SetItemWeeklyWindow(req *types.SetItemWeeklyWindowAPIRequest) (resp *types.ItemWeeklyWindowResponse, err error) {
	return setItemWindow(l.ctx, l.svcCtx, req)
}
