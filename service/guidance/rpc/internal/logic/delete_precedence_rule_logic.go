package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/rules/precedence"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePrecedenceRuleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeletePrecedenceRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePrecedenceRuleLogic {
	return &DeletePrecedenceRuleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DeletePrecedenceRuleLogic) DeletePrecedenceRule(in *v1_guidancev1.DeletePrecedenceRuleRequest) (*v1_guidancev1.DeletePrecedenceRuleResponse, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, precedenceRPCError(precedence.ErrInvalid)
	}
	err = l.svcCtx.PrecedenceManager.Delete(l.ctx, principal, precedence.DeleteInput{
		RuleID: in.RuleId, ExpectedVersion: in.ExpectedVersion, OperationID: in.OperationId,
	})
	if err != nil {
		return nil, precedenceRPCError(err)
	}
	return &v1_guidancev1.DeletePrecedenceRuleResponse{RuleId: in.RuleId, Deleted: true}, nil
}
