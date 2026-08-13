package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type UpdateAccountDisplayProfileLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewUpdateAccountDisplayProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAccountDisplayProfileLogic {
	return &UpdateAccountDisplayProfileLogic{ctx: ctx, svc: svcCtx}
}

func (l *UpdateAccountDisplayProfileLogic) UpdateAccountDisplayProfile(in *identityv1.UpdateAccountDisplayProfileRequest) (*identityv1.AccountDisplayProfile, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := l.svc.Managers.Account.UpdateDisplayProfile(ctx, operator, in.GetNickname())
	if err != nil {
		return nil, accountManagementRPCError(err)
	}
	return accountDisplayProfileResponse(profile), nil
}
