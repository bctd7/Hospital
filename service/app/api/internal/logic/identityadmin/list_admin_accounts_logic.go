package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type ListAdminAccountsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAdminAccountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminAccountsLogic {
	return &ListAdminAccountsLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *ListAdminAccountsLogic) ListAdminAccounts(req *types.ListAdminAccountsRequest) (*types.ListAdminAccountsResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.ListAdminAccounts(ctx, &identityv1.ListAdminAccountsRequest{
		Page: req.Page, PageSize: req.PageSize, Nickname: req.Nickname, IdentityType: req.IdentityType,
		Status: req.Status, DepartmentId: req.DepartmentID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return adminAccountsPageResponse(value), nil
}
