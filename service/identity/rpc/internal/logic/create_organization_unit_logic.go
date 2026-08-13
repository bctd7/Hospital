package logic

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/organization"
	organizationmanager "hospital/service/identity/rpc/internal/organization/manager"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateOrganizationUnitLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateOrganizationUnitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrganizationUnitLogic {
	return &CreateOrganizationUnitLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateOrganizationUnitLogic) CreateOrganizationUnit(in *identityv1.CreateOrganizationUnitRequest) (*identityv1.AdminOrganizationUnit, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	unit, err := l.svcCtx.Managers.OrganizationUnit.CreateUnit(ctx, operator, organizationmanager.CreateUnitCommand{
		Type:        organization.UnitType(in.GetUnitType()),
		ParentID:    in.GetParentId(),
		Name:        in.GetName(),
		OperationID: in.GetOperationId(),
		RequestID:   in.GetRequestId(),
	})
	if err != nil {
		logging.Error(ctx, organizationmanager.ActionUnitCreated, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("parent_id", in.GetParentId()),
			logx.Field("operation_id", in.GetOperationId()))
		return nil, organizationRPCError(err)
	}
	logging.Info(ctx, organizationmanager.ActionUnitCreated,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("unit_id", unit.ID),
		logx.Field("operation_id", in.GetOperationId()))
	return adminOrganizationUnitResponse(unit), nil
}
