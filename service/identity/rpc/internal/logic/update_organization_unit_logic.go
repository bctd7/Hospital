package logic

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateOrganizationUnitLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateOrganizationUnitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateOrganizationUnitLogic {
	return &UpdateOrganizationUnitLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateOrganizationUnitLogic) UpdateOrganizationUnit(in *identityv1.UpdateOrganizationUnitRequest) (*identityv1.AdminOrganizationUnit, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	unit, err := l.svcCtx.Managers.OrganizationUnit.UpdateUnit(ctx, operator, organizationmanager.UpdateUnitCommand{
		UnitID:          in.GetUnitId(),
		Name:            in.Name,
		ParentID:        in.ParentId,
		ExpectedVersion: in.GetVersion(),
		OperationID:     in.GetOperationId(),
		RequestID:       in.GetRequestId(),
	})
	if err != nil {
		logging.Error(ctx, organizationmanager.ActionUnitUpdated, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("unit_id", in.GetUnitId()),
			logx.Field("operation_id", in.GetOperationId()))
		return nil, organizationRPCError(err)
	}
	logging.Info(ctx, organizationmanager.ActionUnitUpdated,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("unit_id", unit.ID),
		logx.Field("operation_id", in.GetOperationId()))
	return adminOrganizationUnitResponse(unit), nil
}
