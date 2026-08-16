// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidanceplanning

import (
	"context"

	"hospital/common/observability/logging"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateSmartAppointmentPlansLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGenerateSmartAppointmentPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateSmartAppointmentPlansLogic {
	return &GenerateSmartAppointmentPlansLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GenerateSmartAppointmentPlansLogic) GenerateSmartAppointmentPlans(req *types.GenerateSmartAppointmentPlansRequest) (resp *types.SmartAppointmentPlansResponse, err error) {
	rpcCtx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Guidance.GenerateSmartAppointmentPlans(rpcCtx, &guidancev1.GenerateSmartAppointmentPlansRequest{
		ItemIds: req.ItemIDs, CandidateDates: req.CandidateDates, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return smartPlansResponse(value), nil
}
