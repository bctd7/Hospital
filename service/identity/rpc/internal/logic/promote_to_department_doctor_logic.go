package logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/svc"
)

type PromoteToDepartmentDoctorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPromoteToDepartmentDoctorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PromoteToDepartmentDoctorLogic {
	return &PromoteToDepartmentDoctorLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PromoteToDepartmentDoctorLogic) PromoteToDepartmentDoctor(in *identityv1.PromoteToDepartmentDoctorRequest) (*identityv1.AuthorizationContext, error) {
	ctx := authorizationRequestContext(l.ctx, in.GetRequestId())
	operator, err := authorizationOperator(ctx)
	if err != nil {
		return nil, err
	}
	principal, err := l.svcCtx.AuthorizationManager.PromoteToDepartmentDoctor(
		ctx, operator.AccountID, in.GetTargetAccountId(), in.GetDepartmentId(),
		in.GetOfflineVerified(), in.GetOperationId(), in.GetRequestId(),
	)
	if err != nil {
		logging.Error(ctx, authorization.ActionDoctorPromoted, err,
			logx.Field("operator_account_id", operator.AccountID),
			logx.Field("target_account_id", in.GetTargetAccountId()),
			logx.Field("department_id", in.GetDepartmentId()))
		return nil, authorizationRPCError(err)
	}
	logging.Info(ctx, authorization.ActionDoctorPromoted,
		logx.Field("operator_account_id", operator.AccountID),
		logx.Field("target_account_id", in.GetTargetAccountId()),
		logx.Field("department_id", in.GetDepartmentId()))
	return authorizationContextResponse(principal), nil
}
