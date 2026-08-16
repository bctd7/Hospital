package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTodayExaminationRecommendationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTodayExaminationRecommendationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTodayExaminationRecommendationLogic {
	return &GetTodayExaminationRecommendationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTodayExaminationRecommendationLogic) GetTodayExaminationRecommendation(in *v1_guidancev1.GetTodayExaminationRecommendationRequest) (*v1_guidancev1.TodayExaminationRecommendation, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	recommendation, err := l.svcCtx.PlanningManager.Today(l.ctx, principal)
	if err != nil {
		return nil, planningRPCError(err)
	}
	stages := make([]*v1_guidancev1.TodayRecommendationStage, 0, len(recommendation.Stages))
	for _, stage := range recommendation.Stages {
		items := make([]*v1_guidancev1.SmartAppointmentPlanItem, 0, len(stage.Items))
		for _, item := range stage.Items {
			items = append(items, planItemResponse(item))
		}
		stages = append(stages, &v1_guidancev1.TodayRecommendationStage{StageNo: stage.StageNo, Title: stage.Title, Status: stage.Status, Focus: stage.Focus, Items: items})
	}
	return &v1_guidancev1.TodayExaminationRecommendation{ServiceDate: recommendation.ServiceDate, Stages: stages, UpdatedAt: formatGuidanceTime(recommendation.UpdatedAt)}, nil
}
