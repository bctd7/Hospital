package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type SearchAdminAccountByPhoneLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewSearchAdminAccountByPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchAdminAccountByPhoneLogic {
	return &SearchAdminAccountByPhoneLogic{ctx: ctx, svc: svcCtx}
}

func (l *SearchAdminAccountByPhoneLogic) SearchAdminAccountByPhone(in *identityv1.SearchAdminAccountByPhoneRequest) (*identityv1.SearchAdminAccountByPhoneResponse, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, masked, verificationStatus, verificationSource, err := l.svc.IdentityAdminManager.SearchByPhone(ctx, operator, in.GetPhone())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return &identityv1.SearchAdminAccountByPhoneResponse{
		Identity: adminAccountSummaryResponse(account),
		Phone:    &identityv1.PhoneBinding{PhoneMasked: masked, VerificationStatus: verificationStatus, VerificationSource: verificationSource},
	}, nil
}
