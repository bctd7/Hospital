package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type RevokeDoctorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDoctorLogic {
	return &RevokeDoctorLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *RevokeDoctorLogic) RevokeDoctor(req *types.AccountMutationRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.RevokeDoctor(ctx, accountMutationRequest(req, logging.RequestIDFromContext(l.ctx)))
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
