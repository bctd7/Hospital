// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidanceconfiguration

import (
	"context"

	"hospital/common/observability/logging"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewPreparationRulesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewPreparationRulesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewPreparationRulesLogic {
	return &PreviewPreparationRulesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewPreparationRulesLogic) PreviewPreparationRules(req *types.PreviewPreparationRulesRequest) (resp *types.PreparationRulePreviewResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Guidance.PreviewPreparationRules(ctx, &guidancev1.PreviewPreparationRulesRequest{
		Description: req.Description, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.PreparationRulePreviewResponse{
		Description: value.GetDescription(), PreparationRules: preparationRuleResponses(value.GetPreparationRules()),
		Reminders: reminderResponses(value.GetReminders()), UnresolvedFragments: value.GetUnresolvedFragments(),
	}, nil
}
