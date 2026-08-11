package identityprofile

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type UpdateAccountDisplayProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAccountDisplayProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAccountDisplayProfileLogic {
	return &UpdateAccountDisplayProfileLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateAccountDisplayProfileLogic) UpdateAccountDisplayProfile(req *types.UpdateAccountDisplayProfileRequest) (*types.AccountDisplayProfileResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.UpdateAccountDisplayProfile(ctx, &identityv1.UpdateAccountDisplayProfileRequest{Nickname: req.Nickname, RequestId: logging.RequestIDFromContext(l.ctx)})
	if err != nil {
		return nil, err
	}
	return &types.AccountDisplayProfileResponse{Nickname: value.Nickname, ManagementVersion: value.GetManagementVersion()}, nil
}
