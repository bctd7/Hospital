package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	accountmanager "hospital/service/identity/rpc/internal/account/manager"
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
	account, actions, err := l.svc.Managers.Account.PromoteDoctor(ctx, operator,
		in.GetAccountId(), in.GetDepartmentId(), accountmanager.DoctorProfileInput{
			DisplayName: in.GetDisplayName(), StaffNo: in.GetStaffNo(), AvatarURL: in.GetAvatarUrl(), Description: in.GetDescription(),
		}, in.GetManagementVersion(), in.GetOfflineVerified(), in.GetOperationId(), in.GetRequestId())
	if err != nil {
		return nil, accountManagementRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
