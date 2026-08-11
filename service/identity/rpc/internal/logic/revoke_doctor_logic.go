package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type RevokeDoctorLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewRevokeDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDoctorLogic {
	return &RevokeDoctorLogic{ctx: ctx, svc: svcCtx}
}

func (l *RevokeDoctorLogic) RevokeDoctor(in *identityv1.AccountMutationRequest) (*identityv1.AdminAccountDetail, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	account, actions, err := l.svc.IdentityAdminManager.RevokeDoctor(ctx, operator, in.GetAccountId(), in.GetManagementVersion(), in.GetOperationId(), in.GetRequestId())
	if err != nil {
		return nil, identityAdminRPCError(err)
	}
	return adminAccountDetailResponse(account, actions), nil
}
