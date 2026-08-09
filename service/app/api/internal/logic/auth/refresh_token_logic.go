// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"

	"google.golang.org/grpc/metadata"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenRequest) (resp *types.TokenResponse, err error) {
	ctx := metadata.AppendToOutgoingContext(
		l.ctx, "x-request-id", logging.RequestIDFromContext(l.ctx),
	)
	pair, err := l.svcCtx.Identity.RefreshAccessToken(ctx, &identityv1.RefreshAccessTokenRequest{
		RefreshToken: req.RefreshToken,
		RequestId:    logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.TokenResponse{
		AccessToken: pair.GetAccessToken(), RefreshToken: pair.GetRefreshToken(),
		AccessExpiresInSeconds:  pair.GetAccessExpiresInSeconds(),
		RefreshExpiresInSeconds: pair.GetRefreshExpiresInSeconds(),
	}, nil
}
