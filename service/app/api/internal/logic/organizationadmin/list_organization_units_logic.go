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

type ListOrganizationUnitsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListOrganizationUnitsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrganizationUnitsLogic {
	return &ListOrganizationUnitsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListOrganizationUnitsLogic) ListOrganizationUnits(req *types.ListOrganizationUnitsRequest) (resp *types.ListOrganizationUnitsResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.ListOrganizationUnits(ctx, &identityv1.ListOrganizationUnitsRequest{
		UnitType:  req.UnitType,
		ParentId:  req.ParentID,
		Status:    req.Status,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return adminOrganizationUnitsResponse(value.GetItems()), nil
}
