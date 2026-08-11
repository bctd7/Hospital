package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type EnableManagedAccountLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewEnableManagedAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableManagedAccountLogic {
	return &EnableManagedAccountLogic{ctx: ctx, svc: svcCtx}
}

func (l *EnableManagedAccountLogic) EnableManagedAccount(in *identityv1.ManagedAccountMutationRequest) (*identityv1.AdminAccountDetail, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, actions, err := l.svc.IdentityAdminManager.SetAccountEnabled(ctx, operator, in.GetAccountId(), true, in.GetManagementVersion(), in.GetOperationId(), in.GetRequestId())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
