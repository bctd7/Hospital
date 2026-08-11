package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/identityadmin"
	"hospital/service/identity/rpc/internal/svc"
)

type PromoteDoctorLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewPromoteDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PromoteDoctorLogic {
	return &PromoteDoctorLogic{ctx: ctx, svc: svcCtx}
}

func (l *PromoteDoctorLogic) PromoteDoctor(in *identityv1.PromoteDoctorRequest) (*identityv1.AdminAccountDetail, error) {
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
