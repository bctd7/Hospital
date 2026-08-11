package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/identityadmin"
	"hospital/service/identity/rpc/internal/svc"
)

type UpdateDoctorLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewUpdateDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDoctorLogic {
	return &UpdateDoctorLogic{ctx: ctx, svc: svcCtx}
}

func (l *UpdateDoctorLogic) UpdateDoctor(in *identityv1.UpdateDoctorRequest) (*identityv1.AdminAccountDetail, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, actions, err := l.svc.IdentityAdminManager.UpdateDoctor(ctx, operator, in.GetAccountId(),
		identityadmin.OptionalDoctorProfileInput{DisplayName: in.DisplayName, StaffNo: in.StaffNo, AvatarURL: in.AvatarUrl, Description: in.Description},
		in.GetManagementVersion(), in.GetOperationId(), in.GetRequestId())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
