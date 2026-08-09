package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/svc"
)

type AssignRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAssignRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AssignRoleLogic {
	return &AssignRoleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AssignRoleLogic) AssignRole(in *identityv1.AssignRoleRequest) (*identityv1.AuthorizationContext, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	principal, err := l.svcCtx.AuthorizationManager.AssignRole(
		ctx, operator.AccountID, in.GetTargetAccountId(), in.GetRoleCode(), in.GetOperationId(), in.GetRequestId(),
	)
	if err != nil {
		logging.Error(ctx, authorization.ActionRoleAssigned, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("target_account_id", in.GetTargetAccountId()))
		return nil, authorizationRPCError(err)
	}
	logging.Info(ctx, authorization.ActionRoleAssigned,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("target_account_id", in.GetTargetAccountId()),
		logx.Field("role_code", in.GetRoleCode()))
	return authorizationContextResponse(principal), nil
}
