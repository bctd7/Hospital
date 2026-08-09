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

type WeChatLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewWeChatLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *WeChatLoginLogic {
	return &WeChatLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *WeChatLoginLogic) WeChatLogin(req *types.WeChatLoginRequest) (resp *types.TokenResponse, err error) {
	ctx := metadata.AppendToOutgoingContext(l.ctx, "x-request-id", logging.RequestIDFromContext(l.ctx))
	pair, err := l.svcCtx.Identity.WeChatLogin(ctx, &identityv1.WeChatLoginRequest{
		LoginCode: req.LoginCode, RequestId: logging.RequestIDFromContext(l.ctx),
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
