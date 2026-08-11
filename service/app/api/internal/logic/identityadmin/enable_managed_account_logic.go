package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type EnableManagedAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnableManagedAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableManagedAccountLogic {
	return &EnableManagedAccountLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *EnableManagedAccountLogic) EnableManagedAccount(req *types.ManagedAccountMutationRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.EnableManagedAccount(ctx, managedAccountMutationRequest(req, logging.RequestIDFromContext(l.ctx)))
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
