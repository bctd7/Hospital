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

type UpdatePrecedenceRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdatePrecedenceRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePrecedenceRuleLogic {
	return &UpdatePrecedenceRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdatePrecedenceRuleLogic) UpdatePrecedenceRule(req *types.UpdatePrecedenceRuleRequest) (resp *types.PrecedenceRuleResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	rule, err := l.svcCtx.Guidance.UpdatePrecedenceRule(ctx, &guidancev1.UpdatePrecedenceRuleRequest{
		RuleId: req.RuleID, StaffReason: req.StaffReason, PatientMessage: req.PatientMessage,
		ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	value := precedenceRuleResponse(rule)
	return &value, nil
}
