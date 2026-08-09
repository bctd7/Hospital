// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchAccountByPhoneLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchAccountByPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchAccountByPhoneLogic {
	return &SearchAccountByPhoneLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchAccountByPhoneLogic) SearchAccountByPhone(req *types.SearchAccountByPhoneRequest) (resp *types.SearchAccountByPhoneResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	lookup, err := l.svcCtx.Identity.FindAccountByPhone(ctx, &identityv1.FindAccountByPhoneRequest{
		Phone: req.Phone, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.SearchAccountByPhoneResponse{
		Identity: identityResponse(lookup.GetAuthorization()),
		Phone: types.PhoneBindingResponse{
			PhoneMasked:        lookup.GetPhone().GetPhoneMasked(),
			VerificationStatus: lookup.GetPhone().GetVerificationStatus(),
			VerificationSource: lookup.GetPhone().GetVerificationSource(),
		},
	}, nil
}
