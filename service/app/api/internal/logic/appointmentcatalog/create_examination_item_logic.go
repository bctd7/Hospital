// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentcatalog

import (
	"context"

	"hospital/common/observability/logging"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateExaminationItemLogic {
	return &CreateExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateExaminationItemLogic) CreateExaminationItem(req *types.CreateExaminationItemRequest) (resp *types.ExaminationItemResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.Appointment.CreateExaminationItem(ctx, &appointmentv1.CreateExaminationItemRequest{
		ExaminationItem: &appointmentv1.ExaminationItemInput{
			OwnerDepartmentId:        req.OwnerDepartmentID,
			Name:                     req.Name,
			Description:              req.Description,
			EstimatedDurationMinutes: req.EstimatedDurationMinutes,
		},
		OperationId: req.OperationID,
		RequestId:   logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return examinationItemResponse(item), nil
}
