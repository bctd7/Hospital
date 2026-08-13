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

type ListExaminationItemsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListExaminationItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListExaminationItemsLogic {
	return &ListExaminationItemsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListExaminationItemsLogic) ListExaminationItems(req *types.ListExaminationItemsRequest) (resp *types.ListExaminationItemsResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.ListExaminationItems(ctx, &appointmentv1.ListExaminationItemsRequest{
		OwnerDepartmentId: req.OwnerDepartmentID,
		Status:            req.Status,
		Page:              req.Page,
		PageSize:          req.PageSize,
		RequestId:         logging.RequestIDFromContext(l.ctx),
		Audience:          "staff",
	})
	if err != nil {
		return nil, err
	}
	return examinationItemsResponse(value), nil
}
