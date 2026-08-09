package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type RevokeRefreshTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeRefreshTokenLogic {
	return &RevokeRefreshTokenLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RevokeRefreshTokenLogic) RevokeRefreshToken(in *identityv1.RevokeRefreshTokenRequest) (*identityv1.RevokeRefreshTokenResponse, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	if err := l.svcCtx.SessionManager.Revoke(ctx, in.GetRefreshToken()); err != nil {
		logging.Error(ctx, "identity.session.revoke.failed", err)
		return nil, sessionRPCError(err)
	}
	logging.Info(ctx, "identity.session.revoked")
	return &identityv1.RevokeRefreshTokenResponse{Revoked: true}, nil
}
