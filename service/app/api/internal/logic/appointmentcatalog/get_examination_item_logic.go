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

type GetExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemLogic {
	return &GetExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetExaminationItemLogic) GetExaminationItem(req *types.ExaminationItemPathRequest) (resp *types.ExaminationItemResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.Appointment.GetExaminationItem(ctx, &appointmentv1.GetExaminationItemRequest{
		ItemId: req.ItemID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return examinationItemResponse(item), nil
}
