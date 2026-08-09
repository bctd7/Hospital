// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetMyPhoneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetMyPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetMyPhoneLogic {
	return &SetMyPhoneLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetMyPhoneLogic) SetMyPhone(req *types.SetPhoneRequest) (resp *types.PhoneBindingResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	binding, err := l.svcCtx.Identity.SetMyPhone(ctx, &identityv1.SetMyPhoneRequest{
		Phone: req.Phone, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.PhoneBindingResponse{
		PhoneMasked: binding.GetPhoneMasked(), VerificationStatus: binding.GetVerificationStatus(),
		VerificationSource: binding.GetVerificationSource(),
	}, nil
}
