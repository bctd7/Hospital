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

type RevokeTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeTokenLogic {
	return &RevokeTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeTokenLogic) RevokeToken(req *types.RefreshTokenRequest) (resp *types.RevokeTokenResponse, err error) {
	ctx := metadata.AppendToOutgoingContext(
		l.ctx, "x-request-id", logging.RequestIDFromContext(l.ctx),
	)
	result, err := l.svcCtx.Identity.RevokeRefreshToken(ctx, &identityv1.RevokeRefreshTokenRequest{
		RefreshToken: req.RefreshToken,
		RequestId:    logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.RevokeTokenResponse{Revoked: result.GetRevoked()}, nil
}
