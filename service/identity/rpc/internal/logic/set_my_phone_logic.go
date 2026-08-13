package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type SetMyPhoneLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetMyPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetMyPhoneLogic {
	return &SetMyPhoneLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetMyPhoneLogic) SetMyPhone(in *identityv1.SetMyPhoneRequest) (*identityv1.PhoneBinding, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	principal, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	binding, err := l.svcCtx.Managers.Authentication.SetMyPhone(ctx, principal.AccountID, in.GetPhone())
	if err != nil {
		logging.Error(ctx, "identity.phone.self_report_failed", err,
			logx.Field("account_id", principal.AccountID))
		return nil, accountRPCError(err)
	}
	logging.Info(ctx, "identity.phone.self_reported",
		logx.Field("account_id", principal.AccountID))
	return &identityv1.PhoneBinding{
		PhoneMasked: binding.Masked, VerificationStatus: binding.VerificationStatus,
		VerificationSource: binding.VerificationSource,
	}, nil
}
