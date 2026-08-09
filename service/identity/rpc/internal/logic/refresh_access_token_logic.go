package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type RefreshAccessTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefreshAccessTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshAccessTokenLogic {
	return &RefreshAccessTokenLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RefreshAccessTokenLogic) RefreshAccessToken(in *identityv1.RefreshAccessTokenRequest) (*identityv1.TokenPair, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	pair, err := l.svcCtx.SessionManager.Refresh(ctx, in.GetRefreshToken())
	if err != nil {
		logging.Error(ctx, "identity.session.refresh.failed", err)
		return nil, sessionRPCError(err)
	}
	logging.Info(ctx, "identity.session.refreshed")
	return tokenPairResponse(pair), nil
}
