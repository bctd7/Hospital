package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type GetAccountDisplayProfileLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewGetAccountDisplayProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountDisplayProfileLogic {
	return &GetAccountDisplayProfileLogic{ctx: ctx, svc: svcCtx}
}

func (l *GetAccountDisplayProfileLogic) GetAccountDisplayProfile(in *identityv1.GetAccountDisplayProfileRequest) (*identityv1.AccountDisplayProfile, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	profile, err := l.svc.IdentityAdminManager.GetDisplayProfile(ctx, operator)
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return accountDisplayProfileResponse(profile), nil
}
