package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/svc"
)

type ChangeAccountStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChangeAccountStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeAccountStatusLogic {
	return &ChangeAccountStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChangeAccountStatusLogic) ChangeAccountStatus(in *identityv1.ChangeAccountStatusRequest) (*identityv1.AuthorizationContext, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	principal, err := l.svcCtx.AuthorizationManager.ChangeAccountStatus(
		ctx, operator, in.GetTargetAccountId(), in.GetStatus(), in.GetOperationId(), in.GetRequestId(),
	)
	if err != nil {
		logging.Error(ctx, authorization.ActionAccountStatusChanged, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("target_account_id", in.GetTargetAccountId()))
		return nil, authorizationRPCError(err)
	}
	logging.Info(ctx, authorization.ActionAccountStatusChanged,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("target_account_id", in.GetTargetAccountId()),
		logx.Field("status", in.GetStatus()))
	return authorizationContextResponse(principal), nil
}
