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

type GetTodayExaminationRecommendationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetTodayExaminationRecommendationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTodayExaminationRecommendationLogic {
	return &GetTodayExaminationRecommendationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTodayExaminationRecommendationLogic) GetTodayExaminationRecommendation() (resp *types.TodayExaminationRecommendationResponse, err error) {
	rpcCtx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Guidance.GetTodayExaminationRecommendation(rpcCtx, &guidancev1.GetTodayExaminationRecommendationRequest{RequestId: logging.RequestIDFromContext(l.ctx)})
	if err != nil {
		return nil, err
	}
	return todayRecommendationResponse(value), nil
}
