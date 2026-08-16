// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidancerules

import (
	"context"

	"hospital/common/observability/logging"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPrecedenceRulesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListPrecedenceRulesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPrecedenceRulesLogic {
	return &ListPrecedenceRulesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListPrecedenceRulesLogic) ListPrecedenceRules(req *types.ListPrecedenceRulesRequest) (resp *types.ListPrecedenceRulesResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.Guidance.ListPrecedenceRules(ctx, &guidancev1.ListPrecedenceRulesRequest{
		ItemId: req.ItemID, Direction: req.Direction, IncludeInferred: req.IncludeInferred,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	response := &types.ListPrecedenceRulesResponse{Rules: make([]types.PrecedenceRuleResponse, 0, len(result.Rules))}
	for _, rule := range result.Rules {
		response.Rules = append(response.Rules, precedenceRuleResponse(rule))
	}
	return response, nil
}
