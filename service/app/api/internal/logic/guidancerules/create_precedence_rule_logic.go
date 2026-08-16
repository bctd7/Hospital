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

type CreatePrecedenceRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreatePrecedenceRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreatePrecedenceRuleLogic {
	return &CreatePrecedenceRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreatePrecedenceRuleLogic) CreatePrecedenceRule(req *types.CreatePrecedenceRuleRequest) (resp *types.PrecedenceRuleResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	rule, err := l.svcCtx.Guidance.CreatePrecedenceRule(ctx, &guidancev1.CreatePrecedenceRuleRequest{
		PredecessorItemId: req.PredecessorItemID,
		SuccessorItemId:   req.SuccessorItemID,
		StaffReason:       req.StaffReason,
		PatientMessage:    req.PatientMessage,
		OperationId:       req.OperationID,
		RequestId:         logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	value := precedenceRuleResponse(rule)
	return &value, nil
}
