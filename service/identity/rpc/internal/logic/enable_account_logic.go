package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type EnableAccountLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewEnableAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableAccountLogic {
	return &EnableAccountLogic{ctx: ctx, svc: svcCtx}
}

func (l *EnableAccountLogic) EnableAccount(in *identityv1.AccountMutationRequest) (*identityv1.AdminAccountDetail, error) {
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
