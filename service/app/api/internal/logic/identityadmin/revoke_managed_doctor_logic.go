package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type RevokeManagedDoctorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeManagedDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeManagedDoctorLogic {
	return &RevokeManagedDoctorLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *RevokeManagedDoctorLogic) RevokeManagedDoctor(req *types.ManagedAccountMutationRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.RevokeManagedDoctor(ctx, managedAccountMutationRequest(req, logging.RequestIDFromContext(l.ctx)))
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
