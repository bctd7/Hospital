package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PreviewPreparationRulesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPreviewPreparationRulesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewPreparationRulesLogic {
	return &PreviewPreparationRulesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PreviewPreparationRulesLogic) PreviewPreparationRules(in *v1_guidancev1.PreviewPreparationRulesRequest) (*v1_guidancev1.PreparationRulePreview, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if err := requireGuidanceStaff(principal); err != nil {
		return nil, err
	}
	preview, err := l.svcCtx.ProjectConfigurationManager.Preview(l.ctx, in.GetDescription())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid preparation description")
	}
	return &v1_guidancev1.PreparationRulePreview{
		Description: preview.Description, PreparationRules: preparationResponses(preview.Rules),
		Reminders: reminderResponses(preview.Reminders), UnresolvedFragments: preview.UnresolvedFragments,
		ParserMode: preview.ParserMode, Warning: preview.Warning,
	}, nil
}
