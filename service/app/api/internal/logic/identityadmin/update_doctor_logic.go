package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"
)

type UpdateDoctorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDoctorLogic {
	return &UpdateDoctorLogic{ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateDoctorLogic) UpdateDoctor(req *types.UpdateDoctorRequest) (*types.AdminAccountDetailResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Identity.UpdateDoctor(ctx, &identityv1.UpdateDoctorRequest{
		AccountId: req.AccountID, DisplayName: req.DisplayName, StaffNo: req.StaffNo,
		AvatarUrl: req.AvatarURL, Description: req.Description,
		ManagementVersion: req.ManagementVersion, OperationId: req.OperationID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return adminAccountDetailResponse(value), nil
}
