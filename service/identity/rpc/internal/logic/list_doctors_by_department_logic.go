package logic

import (
	"context"

	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/svc"
)

type ListDoctorsByDepartmentLogic struct {
	ctx context.Context
	svc *svc.ServiceContext
}

func NewListDoctorsByDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDoctorsByDepartmentLogic {
	return &ListDoctorsByDepartmentLogic{ctx: ctx, svc: svcCtx}
}

func (l *ListDoctorsByDepartmentLogic) ListDoctorsByDepartment(in *identityv1.ListDoctorsByDepartmentRequest) (*identityv1.ListDoctorsByDepartmentResponse, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	page, err := l.svc.IdentityAdminManager.ListDoctors(ctx, in.GetDepartmentId(), in.GetPage(), in.GetPageSize())
	if err != nil {
		return nil, identityDirectoryRPCError(err)
	}
	return doctorsPageResponse(page), nil
}
