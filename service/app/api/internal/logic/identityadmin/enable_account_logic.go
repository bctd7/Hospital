package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type EnableAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewEnableAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableAccountLogic {
	return &EnableAccountLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *EnableAccountLogic) EnableAccount(req *types.AccountMutationRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.EnableAccount(ctx, accountMutationRequest(req, logging.RequestIDFromContext(l.ctx)))
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
