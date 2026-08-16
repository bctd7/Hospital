package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/planning"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmSmartAppointmentPlanLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmSmartAppointmentPlanLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmSmartAppointmentPlanLogic {
	return &ConfirmSmartAppointmentPlanLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ConfirmSmartAppointmentPlanLogic) ConfirmSmartAppointmentPlan(in *v1_guidancev1.ConfirmSmartAppointmentPlanRequest) (*v1_guidancev1.ConfirmedSmartAppointmentPlan, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	ids, err := l.svcCtx.PlanningManager.Confirm(l.ctx, principal, planning.ConfirmCommand{PlanID: in.GetPlanId(), OperationID: in.GetOperationId(), PatientDisplayName: in.GetPatientDisplayName(), PatientPhoneMasked: in.GetPatientPhoneMasked(), RequestID: in.GetRequestId()})
	if err != nil {
		return nil, planningRPCError(err)
	}
	return &v1_guidancev1.ConfirmedSmartAppointmentPlan{PlanId: in.GetPlanId(), BookingIds: ids}, nil
}
