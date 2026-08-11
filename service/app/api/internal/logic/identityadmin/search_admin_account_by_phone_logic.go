package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type SearchAdminAccountByPhoneLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchAdminAccountByPhoneLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchAdminAccountByPhoneLogic {
	return &SearchAdminAccountByPhoneLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *SearchAdminAccountByPhoneLogic) SearchAdminAccountByPhone(req *types.AdminSearchAccountByPhoneRequest) (*types.AdminSearchAccountByPhoneResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.SearchAdminAccountByPhone(ctx, &identityv1.SearchAdminAccountByPhoneRequest{Phone: req.Phone, RequestId: logging.RequestIDFromContext(l.ctx)})
	if err != nil {
		return nil, err
	}
	return &types.AdminSearchAccountByPhoneResponse{Identity: *adminAccountSummaryResponse(value.GetIdentity()), Phone: types.PhoneBindingResponse{PhoneMasked: value.GetPhone().GetPhoneMasked(), VerificationStatus: value.GetPhone().GetVerificationStatus(), VerificationSource: value.GetPhone().GetVerificationSource()}}, nil
}
