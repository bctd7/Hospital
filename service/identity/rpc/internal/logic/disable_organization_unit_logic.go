package logic

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableOrganizationUnitLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableOrganizationUnitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableOrganizationUnitLogic {
	return &DisableOrganizationUnitLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisableOrganizationUnitLogic) DisableOrganizationUnit(in *identityv1.ChangeOrganizationUnitStatusRequest) (*identityv1.AdminOrganizationUnit, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	unit, err := l.svcCtx.Managers.OrganizationUnit.DisableUnit(ctx, operator, organizationmanager.ChangeUnitStatusCommand{
		UnitID:          in.GetUnitId(),
		ExpectedVersion: in.GetVersion(),
		OperationID:     in.GetOperationId(),
		RequestID:       in.GetRequestId(),
	})
	if err != nil {
		logging.Error(ctx, organizationmanager.ActionUnitDisabled, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("unit_id", in.GetUnitId()),
			logx.Field("operation_id", in.GetOperationId()))
		return nil, organizationRPCError(err)
	}
	logging.Info(ctx, organizationmanager.ActionUnitDisabled,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("unit_id", unit.ID),
		logx.Field("operation_id", in.GetOperationId()))
	return adminOrganizationUnitResponse(unit), nil
}
