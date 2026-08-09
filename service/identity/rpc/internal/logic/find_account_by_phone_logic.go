package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type FindAccountByPhoneLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFindAccountByPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FindAccountByPhoneLogic {
	return &FindAccountByPhoneLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FindAccountByPhoneLogic) FindAccountByPhone(in *identityv1.FindAccountByPhoneRequest) (*identityv1.AccountLookup, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	lookup, err := l.svcCtx.AccountManager.FindByPhone(ctx, operator, in.GetPhone())
	if err != nil {
		logging.Error(ctx, "identity.account.phone_search_failed", err,
			logx.Field("operator_account_id", operator.AccountID))
		return nil, accountRPCError(err)
	}
	logging.Info(ctx, "identity.account.phone_searched",
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("target_account_id", lookup.Principal.AccountID))
	return &identityv1.AccountLookup{
		Authorization: authorizationContextResponse(lookup.Principal),
		Phone: &identityv1.PhoneBinding{
			PhoneMasked: lookup.Phone.Masked, VerificationStatus: lookup.Phone.VerificationStatus,
			VerificationSource: lookup.Phone.VerificationSource,
		},
	}, nil
}
