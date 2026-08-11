package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type ChangeManagedDoctorDepartmentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangeManagedDoctorDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeManagedDoctorDepartmentLogic {
	return &ChangeManagedDoctorDepartmentLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *ChangeManagedDoctorDepartmentLogic) ChangeManagedDoctorDepartment(req *types.ChangeManagedDoctorDepartmentRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.ChangeManagedDoctorDepartment(ctx, &identityv1.ChangeManagedDoctorDepartmentRequest{AccountId: req.AccountID, DepartmentId: req.DepartmentID, ManagementVersion: req.ManagementVersion, OperationId: req.OperationID, RequestId: logging.RequestIDFromContext(l.ctx)})
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
