package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/identityadmin"
	"hospital/service/identity/rpc/internal/svc"
)

type ListAdminAccountsLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewListAdminAccountsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAdminAccountsLogic {
	return &ListAdminAccountsLogic{ctx: ctx, svc: svcCtx}
}

func (l *ListAdminAccountsLogic) ListAdminAccounts(in *identityv1.ListAdminAccountsRequest) (*identityv1.ListAdminAccountsResponse, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	page, err := l.svc.IdentityAdminManager.ListAccounts(ctx, operator, identityadmin.AccountFilter{
		Page: in.GetPage(), PageSize: in.GetPageSize(), Nickname: in.GetNickname(),
		IdentityType: in.GetIdentityType(), Status: in.GetStatus(), DepartmentID: in.GetDepartmentId(),
	})
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountsPageResponse(page), nil
}
