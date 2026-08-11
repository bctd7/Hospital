package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type GetAdminAccountLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewGetAdminAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminAccountLogic {
	return &GetAdminAccountLogic{ctx: ctx, svc: svcCtx}
}

func (l *GetAdminAccountLogic) GetAdminAccount(in *identityv1.GetAdminAccountRequest) (*identityv1.AdminAccountDetail, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, actions, err := l.svc.IdentityAdminManager.GetAccount(ctx, operator, in.GetAccountId())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
