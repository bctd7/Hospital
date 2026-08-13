// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentcatalog

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableExaminationItemLogic {
	return &DisableExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableExaminationItemLogic) DisableExaminationItem(req *types.ChangeExaminationItemStatusRequest) (resp *types.ExaminationItemResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.Appointment.DisableExaminationItem(
		ctx, changeStatusRequest(req, logging.RequestIDFromContext(l.ctx)),
	)
	if err != nil {
		return nil, err
	}
	return examinationItemResponse(item), nil
}
