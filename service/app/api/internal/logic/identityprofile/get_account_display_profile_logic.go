package identityprofile

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type GetAccountDisplayProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAccountDisplayProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAccountDisplayProfileLogic {
	return &GetAccountDisplayProfileLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *GetAccountDisplayProfileLogic) GetAccountDisplayProfile() (*types.AccountDisplayProfileResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.GetAccountDisplayProfile(ctx, &identityv1.GetAccountDisplayProfileRequest{RequestId: logging.RequestIDFromContext(l.ctx)})
	if err != nil {
		return nil, err
	}
	return &types.AccountDisplayProfileResponse{Nickname: value.Nickname, ManagementVersion: value.GetManagementVersion()}, nil
}
