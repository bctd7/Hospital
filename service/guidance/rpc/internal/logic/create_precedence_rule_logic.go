package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/rules/precedence"
	precedencemanager "hospital/service/guidance/rpc/internal/rules/precedence/manager"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreatePrecedenceRuleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreatePrecedenceRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePrecedenceRuleLogic {
	return &CreatePrecedenceRuleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreatePrecedenceRuleLogic) CreatePrecedenceRule(in *v1_guidancev1.CreatePrecedenceRuleRequest) (*v1_guidancev1.PrecedenceRule, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, precedenceRPCError(precedence.ErrInvalid)
	}
	rule, err := l.svcCtx.PrecedenceManager.Create(l.ctx, principal, precedencemanager.CreateInput{
		PredecessorItemID: in.PredecessorItemId,
		SuccessorItemID:   in.SuccessorItemId,
		StaffReason:       in.StaffReason,
		PatientMessage:    in.PatientMessage,
		OperationID:       in.OperationId,
	})
	if err != nil {
		return nil, precedenceRPCError(err)
	}
	return directRuleResponse(rule), nil
}
