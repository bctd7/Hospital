package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type ChangeDoctorDepartmentLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewChangeDoctorDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangeDoctorDepartmentLogic {
	return &ChangeDoctorDepartmentLogic{ctx: ctx, svc: svcCtx}
}

func (l *ChangeDoctorDepartmentLogic) ChangeDoctorDepartment(in *identityv1.ChangeDoctorDepartmentRequest) (*identityv1.AdminAccountDetail, error) {
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
