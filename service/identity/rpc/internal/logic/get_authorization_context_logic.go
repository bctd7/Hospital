package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type GetAuthorizationContextLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthorizationContextLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthorizationContextLogic {
	return &GetAuthorizationContextLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAuthorizationContextLogic) GetAuthorizationContext(in *identityv1.GetAuthorizationContextRequest) (*identityv1.AuthorizationContext, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	principal, err := l.svcCtx.Managers.Authorization.GetAuthorizationContext(ctx, operator, in.GetAccountId())
	if err != nil {
		logging.Error(ctx, "identity.authorization.context.read", err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("target_account_id", in.GetAccountId()))
		return nil, authorizationRPCError(err)
	}
	return authorizationContextResponse(principal), nil
}
