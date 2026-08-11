package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type GetAdminAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAdminAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminAccountLogic {
	return &GetAdminAccountLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *GetAdminAccountLogic) GetAdminAccount(req *types.AdminAccountPathRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.GetAdminAccount(ctx, &identityv1.GetAdminAccountRequest{AccountId: req.AccountID, RequestId: logging.RequestIDFromContext(l.ctx)})
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
