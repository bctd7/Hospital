package organizationdirectory

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type ListDoctorsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListDoctorsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDoctorsLogic {
	return &ListDoctorsLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *ListDoctorsLogic) ListDoctors(req *types.DirectoryDoctorPathRequest) (*types.ListDoctorsResponse, error) {
	value, err := l.svcCtx.Identity.ListDoctorsByDepartment(l.ctx, &identityv1.ListDoctorsByDepartmentRequest{
		DepartmentId: req.DepartmentID, Page: req.Page, PageSize: req.PageSize,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return doctorsResponse(value), nil
}
