package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type PromoteManagedDoctorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPromoteManagedDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PromoteManagedDoctorLogic {
	return &PromoteManagedDoctorLogic{ctx: ctx, svcCtx: svcCtx}
}
func (l *PromoteManagedDoctorLogic) PromoteManagedDoctor(req *types.PromoteManagedDoctorRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.PromoteManagedDoctor(ctx, &identityv1.PromoteManagedDoctorRequest{
		AccountId: req.AccountID, DepartmentId: req.DepartmentID, DisplayName: req.DisplayName,
		StaffNo: req.StaffNo, AvatarUrl: req.AvatarURL, Description: req.Description,
		ManagementVersion: req.ManagementVersion, OfflineVerified: req.OfflineVerified,
		OperationId: req.OperationID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
