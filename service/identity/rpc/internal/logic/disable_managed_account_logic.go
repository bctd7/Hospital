package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type DisableManagedAccountLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewDisableManagedAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableManagedAccountLogic {
	return &DisableManagedAccountLogic{ctx: ctx, svc: svcCtx}
}

func (l *DisableManagedAccountLogic) DisableManagedAccount(in *identityv1.ManagedAccountMutationRequest) (*identityv1.AdminAccountDetail, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, actions, err := l.svc.IdentityAdminManager.SetAccountEnabled(ctx, operator, in.GetAccountId(), false, in.GetManagementVersion(), in.GetOperationId(), in.GetRequestId())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
