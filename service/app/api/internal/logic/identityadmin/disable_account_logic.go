package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type DisableAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDisableAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableAccountLogic {
	return &DisableAccountLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *DisableAccountLogic) DisableAccount(req *types.AccountMutationRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.DisableAccount(ctx, accountMutationRequest(req, logging.RequestIDFromContext(l.ctx)))
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
