// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organizationdirectory

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrganizationContextLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrganizationContextLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrganizationContextLogic {
	return &GetOrganizationContextLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrganizationContextLogic) GetOrganizationContext() (resp *types.OrganizationContextResponse, err error) {
	value, err := l.svcCtx.Identity.GetOrganizationContext(l.ctx, &identityv1.GetOrganizationContextRequest{
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return organizationContextResponse(value), nil
}
