package logic

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PhoneLoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPhoneLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PhoneLoginLogic {
	return &PhoneLoginLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *PhoneLoginLogic) PhoneLogin(in *identityv1.PhoneLoginRequest) (*identityv1.TokenPair, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	pair, err := l.svcCtx.Managers.Authentication.Login(ctx, in.GetPhone(), in.GetVerificationCode())
	if err != nil {
		logging.Error(ctx, "identity.phone_login.failed", err)
		return nil, authenticationRPCError(err)
	}
	logging.Info(ctx, "identity.phone_login.succeeded")
	return tokenPairResponse(pair), nil
}
