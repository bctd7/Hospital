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

type DeletePrecedenceRuleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeletePrecedenceRuleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePrecedenceRuleLogic {
	return &DeletePrecedenceRuleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeletePrecedenceRuleLogic) DeletePrecedenceRule(req *types.DeletePrecedenceRuleRequest) (resp *types.DeletePrecedenceRuleResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.Guidance.DeletePrecedenceRule(ctx, &guidancev1.DeletePrecedenceRuleRequest{
		RuleId: req.RuleID, ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.DeletePrecedenceRuleResponse{RuleID: result.RuleId, Deleted: result.Deleted}, nil
}
