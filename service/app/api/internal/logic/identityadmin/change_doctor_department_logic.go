package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type ChangeDoctorDepartmentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangeDoctorDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeDoctorDepartmentLogic {
	return &ChangeDoctorDepartmentLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *ChangeDoctorDepartmentLogic) ChangeDoctorDepartment(req *types.ChangeDoctorDepartmentRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.ChangeDoctorDepartment(ctx, &identityv1.ChangeDoctorDepartmentRequest{
		AccountId: req.AccountID, DepartmentId: req.DepartmentID,
		ManagementVersion: req.ManagementVersion, OperationId: req.OperationID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
