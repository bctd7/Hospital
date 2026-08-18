// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidanceplanning

import (
	"context"

	"hospital/common/observability/logging"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmSmartAppointmentPlanLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConfirmSmartAppointmentPlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmSmartAppointmentPlanLogic {
	return &ConfirmSmartAppointmentPlanLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConfirmSmartAppointmentPlanLogic) ConfirmSmartAppointmentPlan(req *types.ConfirmSmartAppointmentPlanPathRequest) (resp *types.ConfirmedSmartAppointmentPlanResponse, err error) {
	rpcCtx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	requestID := logging.RequestIDFromContext(l.ctx)
	display, err := currentPatientDisplay(rpcCtx, l.svcCtx, requestID)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Guidance.ConfirmSmartAppointmentPlan(rpcCtx, &guidancev1.ConfirmSmartAppointmentPlanRequest{
		PlanId: req.PlanID, OperationId: req.OperationID, PatientDisplayName: display.name,
		PatientPhoneMasked: display.maskedPhone, RequestId: requestID,
	})
	if err != nil {
		return nil, err
	}
	return &types.ConfirmedSmartAppointmentPlanResponse{PlanID: value.GetPlanId(), BookingIDs: value.GetBookingIds()}, nil
}
