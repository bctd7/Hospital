// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package identityadmin

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PromoteDoctorLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPromoteDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PromoteDoctorLogic {
	return &PromoteDoctorLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PromoteDoctorLogic) PromoteDoctor(req *types.PromoteDoctorRequest) (resp *types.CurrentIdentityResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	principal, err := l.svcCtx.Identity.PromoteToDepartmentDoctor(ctx, &identityv1.PromoteToDepartmentDoctorRequest{
		TargetAccountId: req.AccountID, DepartmentId: req.DepartmentID,
		OfflineVerified: req.OfflineVerified, OperationId: req.OperationID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	response := identityResponse(principal)
	return &response, nil
}
