package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type ChangeManagedDoctorDepartmentLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewChangeManagedDoctorDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeManagedDoctorDepartmentLogic {
	return &ChangeManagedDoctorDepartmentLogic{ctx: ctx, svc: svcCtx}
}

func (l *ChangeManagedDoctorDepartmentLogic) ChangeManagedDoctorDepartment(in *identityv1.ChangeManagedDoctorDepartmentRequest) (*identityv1.AdminAccountDetail, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, actions, err := l.svc.IdentityAdminManager.ChangeDoctorDepartment(ctx, operator,
		in.GetAccountId(), in.GetDepartmentId(), in.GetManagementVersion(), in.GetOperationId(), in.GetRequestId())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
