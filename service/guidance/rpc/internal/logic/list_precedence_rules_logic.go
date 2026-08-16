package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/rules/precedence"
	precedencemanager "hospital/service/guidance/rpc/internal/rules/precedence/manager"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListPrecedenceRulesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPrecedenceRulesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPrecedenceRulesLogic {
	return &ListPrecedenceRulesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListPrecedenceRulesLogic) ListPrecedenceRules(in *v1_guidancev1.ListPrecedenceRulesRequest) (*v1_guidancev1.ListPrecedenceRulesResponse, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, precedenceRPCError(precedence.ErrInvalid)
	}
	values, err := l.svcCtx.PrecedenceManager.List(l.ctx, principal, precedencemanager.ListInput{
		ItemID: in.ItemId, Direction: precedence.Direction(in.Direction), IncludeInferred: in.IncludeInferred,
	})
	if err != nil {
		return nil, precedenceRPCError(err)
	}
	response := &v1_guidancev1.ListPrecedenceRulesResponse{Rules: make([]*v1_guidancev1.PrecedenceRule, 0, len(values))}
	for _, value := range values {
		response.Rules = append(response.Rules, precedenceRuleResponse(value))
	}
	return response, nil
}
