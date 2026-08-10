package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/svc"
)

type ChangeStaffDepartmentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChangeStaffDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeStaffDepartmentLogic {
	return &ChangeStaffDepartmentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ChangeStaffDepartmentLogic) ChangeStaffDepartment(in *identityv1.ChangeStaffDepartmentRequest) (*identityv1.AuthorizationContext, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	principal, err := l.svcCtx.AuthorizationManager.ChangeStaffDepartment(
		ctx, operator, in.GetTargetAccountId(), in.GetDepartmentId(), in.GetOperationId(), in.GetRequestId(),
	)
	if err != nil {
		logging.Error(ctx, authorization.ActionDepartmentChanged, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("target_account_id", in.GetTargetAccountId()))
		return nil, authorizationRPCError(err)
	}
	logging.Info(ctx, authorization.ActionDepartmentChanged,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("target_account_id", in.GetTargetAccountId()),
		logx.Field("department_id", in.GetDepartmentId()))
	return authorizationContextResponse(principal), nil
}
