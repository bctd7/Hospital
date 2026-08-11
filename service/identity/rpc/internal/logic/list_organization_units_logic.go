package logic

import (
	"context"
	"strings"

	"hospital/common/observability/logging"
	"hospital/service/identity/rpc/internal/organization"

	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListOrganizationUnitsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListOrganizationUnitsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListOrganizationUnitsLogic {
	return &ListOrganizationUnitsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Administrator organization management.
func (l *ListOrganizationUnitsLogic) ListOrganizationUnits(in *identityv1.ListOrganizationUnitsRequest) (*identityv1.ListOrganizationUnitsResponse, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	filter := organization.ListFilter{
		Type: organization.UnitType(strings.TrimSpace(in.GetUnitType())),
	}
	if parentID := strings.TrimSpace(in.GetParentId()); parentID != "" {
		filter.ParentID = &parentID
	}
	statusValue := strings.TrimSpace(in.GetStatus())
	switch statusValue {
	case "", string(organization.StatusActive):
		status := organization.StatusActive
		filter.Status = &status
	case "all":
		// nil means no status filter.
	default:
		status := organization.Status(statusValue)
		filter.Status = &status
	}

	units, err := l.svcCtx.OrganizationManager.ListManagedUnits(ctx, operator, filter)
	if err != nil {
		logging.Error(ctx, "identity.organization.units.list", err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("unit_type", in.GetUnitType()),
			logx.Field("parent_id", in.GetParentId()))
		return nil, organizationRPCError(err)
	}
	return adminOrganizationUnitsResponse(units), nil
}
