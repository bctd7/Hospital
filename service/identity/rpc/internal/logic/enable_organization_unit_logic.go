package logic

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/organization"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnableOrganizationUnitLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnableOrganizationUnitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableOrganizationUnitLogic {
	return &EnableOrganizationUnitLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EnableOrganizationUnitLogic) EnableOrganizationUnit(in *identityv1.ChangeOrganizationUnitStatusRequest) (*identityv1.AdminOrganizationUnit, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	unit, err := l.svcCtx.OrganizationManager.EnableUnit(ctx, operator, organization.ChangeUnitStatusCommand{
		UnitID:          in.GetUnitId(),
		ExpectedVersion: in.GetVersion(),
		OperationID:     in.GetOperationId(),
		RequestID:       in.GetRequestId(),
	})
	if err != nil {
		logging.Error(ctx, organization.ActionUnitEnabled, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("unit_id", in.GetUnitId()),
			logx.Field("operation_id", in.GetOperationId()))
		return nil, organizationRPCError(err)
	}
	logging.Info(ctx, organization.ActionUnitEnabled,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("unit_id", unit.ID),
		logx.Field("operation_id", in.GetOperationId()))
	return adminOrganizationUnitResponse(unit), nil
}
