package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/planning"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GenerateSmartAppointmentPlansLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenerateSmartAppointmentPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateSmartAppointmentPlansLogic {
	return &GenerateSmartAppointmentPlansLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GenerateSmartAppointmentPlansLogic) GenerateSmartAppointmentPlans(in *v1_guidancev1.GenerateSmartAppointmentPlansRequest) (*v1_guidancev1.SmartAppointmentPlansResponse, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	availability := make([]planning.CandidateAvailability, 0, len(in.GetCandidateAvailability()))
	for _, value := range in.GetCandidateAvailability() {
		availability = append(availability, planning.CandidateAvailability{ServiceDate: value.GetServiceDate(), Sessions: value.GetSessions()})
	}
	plans, err := l.svcCtx.PlanningManager.Generate(l.ctx, principal, planning.GenerateCommand{
		ItemIDs: in.GetItemIds(), CandidateDates: in.GetCandidateDates(), CandidateAvailability: availability,
	})
	if err != nil {
		return nil, planningRPCError(err)
	}
	response := &v1_guidancev1.SmartAppointmentPlansResponse{Plans: make([]*v1_guidancev1.SmartAppointmentPlan, 0, len(plans))}
	for _, plan := range plans {
		response.Plans = append(response.Plans, smartPlanResponse(plan))
	}
	return response, nil
}
