package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type DisableManagedAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableManagedAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableManagedAccountLogic {
	return &DisableManagedAccountLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *DisableManagedAccountLogic) DisableManagedAccount(req *types.ManagedAccountMutationRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.DisableManagedAccount(ctx, managedAccountMutationRequest(req, logging.RequestIDFromContext(l.ctx)))
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
