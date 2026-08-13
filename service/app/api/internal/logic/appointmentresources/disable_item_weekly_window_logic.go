// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableItemWeeklyWindowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableItemWeeklyWindowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableItemWeeklyWindowLogic {
	return &DisableItemWeeklyWindowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableItemWeeklyWindowLogic) DisableItemWeeklyWindow(req *types.ChangeAppointmentResourceStatusRequest) (resp *types.ItemWeeklyWindowResponse, err error) {
	return disableItemWindow(l.ctx, l.svcCtx, req)
}
