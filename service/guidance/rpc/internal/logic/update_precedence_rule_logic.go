package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/rules/precedence"
	precedencemanager "hospital/service/guidance/rpc/internal/rules/precedence/manager"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdatePrecedenceRuleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdatePrecedenceRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePrecedenceRuleLogic {
	return &UpdatePrecedenceRuleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdatePrecedenceRuleLogic) UpdatePrecedenceRule(in *v1_guidancev1.UpdatePrecedenceRuleRequest) (*v1_guidancev1.PrecedenceRule, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, precedenceRPCError(precedence.ErrInvalid)
	}
	rule, err := l.svcCtx.PrecedenceManager.Update(l.ctx, principal, precedencemanager.UpdateInput{
		RuleID: in.RuleId, StaffReason: in.StaffReason, PatientMessage: in.PatientMessage,
		ExpectedVersion: in.ExpectedVersion, OperationID: in.OperationId,
	})
	if err != nil {
		return nil, precedenceRPCError(err)
	}
	return directRuleResponse(rule), nil
}
