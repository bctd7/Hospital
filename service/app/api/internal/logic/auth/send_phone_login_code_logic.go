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

type SendPhoneLoginCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendPhoneLoginCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendPhoneLoginCodeLogic {
	return &SendPhoneLoginCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendPhoneLoginCodeLogic) SendPhoneLoginCode(req *types.SendPhoneLoginCodeRequest) (resp *types.SendPhoneLoginCodeResponse, err error) {
	ctx := metadata.AppendToOutgoingContext(l.ctx, "x-request-id", logging.RequestIDFromContext(l.ctx))
	result, err := l.svcCtx.Identity.SendPhoneLoginCode(ctx, &identityv1.SendPhoneLoginCodeRequest{
		Phone: req.Phone, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.SendPhoneLoginCodeResponse{
		Accepted: result.GetAccepted(), RetryAfterSeconds: result.GetRetryAfterSeconds(),
	}, nil
}
