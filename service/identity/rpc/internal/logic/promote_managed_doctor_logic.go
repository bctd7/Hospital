package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/identityadmin"
	"hospital/service/identity/rpc/internal/svc"
)

type PromoteManagedDoctorLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewPromoteManagedDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PromoteManagedDoctorLogic {
	return &PromoteManagedDoctorLogic{ctx: ctx, svc: svcCtx}
}

func (l *PromoteManagedDoctorLogic) PromoteManagedDoctor(in *identityv1.PromoteManagedDoctorRequest) (*identityv1.AdminAccountDetail, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, actions, err := l.svc.IdentityAdminManager.PromoteDoctor(ctx, operator,
		in.GetAccountId(), in.GetDepartmentId(), identityadmin.DoctorProfileInput{
			DisplayName: in.GetDisplayName(), StaffNo: in.GetStaffNo(), AvatarURL: in.GetAvatarUrl(), Description: in.GetDescription(),
		}, in.GetManagementVersion(), in.GetOfflineVerified(), in.GetOperationId(), in.GetRequestId())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
