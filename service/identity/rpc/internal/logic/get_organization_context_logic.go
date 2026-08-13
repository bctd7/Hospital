package logic

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrganizationContextLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetOrganizationContextLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrganizationContextLogic {
	return &GetOrganizationContextLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Organization directory shared by visitors and authenticated identities.
func (l *GetOrganizationContextLogic) GetOrganizationContext(in *identityv1.GetOrganizationContextRequest) (*identityv1.OrganizationContext, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	value, err := l.svcCtx.Managers.OrganizationDirectory.GetDirectoryContext(ctx)
	if err != nil {
		logging.Error(ctx, "identity.organization.directory.context.read", err)
		return nil, organizationRPCError(err)
	}
	return organizationContextResponse(value), nil
}
