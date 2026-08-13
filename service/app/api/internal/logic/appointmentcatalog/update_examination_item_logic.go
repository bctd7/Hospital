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

type UpdateExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateExaminationItemLogic {
	return &UpdateExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateExaminationItemLogic) UpdateExaminationItem(req *types.UpdateExaminationItemRequest) (resp *types.ExaminationItemResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.Appointment.UpdateExaminationItem(ctx, &appointmentv1.UpdateExaminationItemRequest{
		ItemId: req.ItemID, Name: req.Name, Description: req.Description,
		ExpectedVersion: req.ExpectedVersion, OperationId: req.OperationID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return examinationItemResponse(item), nil
}
