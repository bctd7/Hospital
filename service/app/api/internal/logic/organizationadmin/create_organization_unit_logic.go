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

type CreateOrganizationUnitLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateOrganizationUnitLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateOrganizationUnitLogic {
	return &CreateOrganizationUnitLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateOrganizationUnitLogic) CreateOrganizationUnit(req *types.CreateOrganizationUnitRequest) (resp *types.AdminOrganizationUnitResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.CreateOrganizationUnit(ctx, &identityv1.CreateOrganizationUnitRequest{
		UnitType: req.UnitType, ParentId: req.ParentID, Name: req.Name,
		OperationId: req.OperationID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return adminOrganizationUnitResponse(value), nil
}
