package logic

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type WeChatLoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewWeChatLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WeChatLoginLogic {
	return &WeChatLoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *WeChatLoginLogic) WeChatLogin(in *identityv1.WeChatLoginRequest) (*identityv1.TokenPair, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	pair, err := l.svcCtx.Managers.Authentication.WeChatLogin(ctx, in.GetLoginCode())
	if err != nil {
		logging.Error(ctx, "identity.wechat.login_failed", err)
		return nil, accountRPCError(err)
	}
	logging.Info(ctx, "identity.wechat.login_succeeded")
	return tokenPairResponse(pair), nil
}
