package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type DisableAccountLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewDisableAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableAccountLogic {
	return &DisableAccountLogic{ctx: ctx, svc: svcCtx}
}

func (l *DisableAccountLogic) DisableAccount(in *identityv1.AccountMutationRequest) (*identityv1.AdminAccountDetail, error) {
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
