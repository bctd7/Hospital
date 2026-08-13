package logic

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendPhoneLoginCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendPhoneLoginCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendPhoneLoginCodeLogic {
	return &SendPhoneLoginCodeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SendPhoneLoginCodeLogic) SendPhoneLoginCode(in *identityv1.SendPhoneLoginCodeRequest) (*identityv1.SendPhoneLoginCodeResponse, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	retryAfter, err := l.svcCtx.Managers.PhoneLogin.SendCode(ctx, in.GetPhone())
	if err != nil {
		logging.Error(ctx, "identity.phone_login.code_send_failed", err)
		return nil, accountRPCError(err)
	}
	logging.Info(ctx, "identity.phone_login.code_sent")
	return &identityv1.SendPhoneLoginCodeResponse{Accepted: true, RetryAfterSeconds: retryAfter}, nil
}
