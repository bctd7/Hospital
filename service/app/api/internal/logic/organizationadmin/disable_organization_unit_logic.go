// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organizationadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableOrganizationUnitLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableOrganizationUnitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableOrganizationUnitLogic {
	return &DisableOrganizationUnitLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableOrganizationUnitLogic) DisableOrganizationUnit(req *types.ChangeOrganizationUnitStatusRequest) (resp *types.AdminOrganizationUnitResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.DisableOrganizationUnit(ctx, &identityv1.ChangeOrganizationUnitStatusRequest{
		UnitId: req.UnitID, Version: req.Version, OperationId: req.OperationID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return adminOrganizationUnitResponse(value), nil
}
