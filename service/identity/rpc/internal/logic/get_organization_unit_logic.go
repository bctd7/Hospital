package logic

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrganizationUnitLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrganizationUnitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrganizationUnitLogic {
	return &GetOrganizationUnitLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetOrganizationUnitLogic) GetOrganizationUnit(in *identityv1.GetOrganizationUnitRequest) (*identityv1.AdminOrganizationUnit, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	unit, err := l.svcCtx.OrganizationManager.GetManagedUnit(ctx, operator, in.GetUnitId())
	if err != nil {
		logging.Error(ctx, "identity.organization.unit.read", err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("unit_id", in.GetUnitId()))
		return nil, organizationRPCError(err)
	}
	return adminOrganizationUnitResponse(unit), nil
}
