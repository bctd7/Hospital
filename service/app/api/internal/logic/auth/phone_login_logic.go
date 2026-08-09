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

type PhoneLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPhoneLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PhoneLoginLogic {
	return &PhoneLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PhoneLoginLogic) PhoneLogin(req *types.PhoneLoginRequest) (resp *types.TokenResponse, err error) {
	ctx := metadata.AppendToOutgoingContext(l.ctx, "x-request-id", logging.RequestIDFromContext(l.ctx))
	pair, err := l.svcCtx.Identity.PhoneLogin(ctx, &identityv1.PhoneLoginRequest{
		Phone: req.Phone, VerificationCode: req.VerificationCode,
		RequestId: logging.RequestIDFromContext(l.ctx),
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
