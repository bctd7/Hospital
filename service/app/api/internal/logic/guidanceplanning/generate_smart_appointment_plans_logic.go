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
	availability := make([]*guidancev1.CandidateAvailability, 0, len(req.CandidateAvailability))
	for _, candidate := range req.CandidateAvailability {
		availability = append(availability, &guidancev1.CandidateAvailability{
			ServiceDate: candidate.ServiceDate,
			Sessions:    candidate.Sessions,
		})
	}
	value, err := l.svcCtx.Guidance.GenerateSmartAppointmentPlans(rpcCtx, &guidancev1.GenerateSmartAppointmentPlansRequest{
		ItemIds: req.ItemIDs, CandidateAvailability: availability, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return smartPlansResponse(value), nil
}
