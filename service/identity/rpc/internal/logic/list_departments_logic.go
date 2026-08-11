package logic

import (
	"context"

	"hospital/common/observability/logging"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDepartmentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListDepartmentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDepartmentsLogic {
	return &ListDepartmentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListDepartmentsLogic) ListDepartments(in *identityv1.ListDepartmentsRequest) (*identityv1.ListDepartmentsResponse, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	departments, err := l.svcCtx.OrganizationManager.ListDirectoryDepartments(ctx, in.GetCampusId())
	if err != nil {
		logging.Error(ctx, "identity.organization.directory.departments.read", err,
			logx.Field("campus_id", in.GetCampusId()))
		return nil, organizationRPCError(err)
	}
	return directoryDepartmentsResponse(departments), nil
}
