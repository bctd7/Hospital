// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentbookings

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyExaminationReportsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMyExaminationReportsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyExaminationReportsLogic {
	return &ListMyExaminationReportsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMyExaminationReportsLogic) ListMyExaminationReports(req *types.ListMyReportsAPIRequest) (resp *types.ListReportsAPIResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.ListMyExaminationReports(rpcCtx, &appointmentv1.ListMyExaminationReportsRequest{Page: req.Page, PageSize: req.PageSize, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	response := &types.ListReportsAPIResponse{Page: value.Page, PageSize: value.PageSize, Total: value.Total, Reports: make([]types.ExaminationReportResponse, 0, len(value.Reports))}
	for _, current := range value.Reports {
		response.Reports = append(response.Reports, *examinationReport(current))
	}
	return response, nil
}
