package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyExaminationReportsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMyExaminationReportsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyExaminationReportsLogic {
	return &ListMyExaminationReportsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListMyExaminationReportsLogic) ListMyExaminationReports(in *appointmentv1.ListMyExaminationReportsRequest) (*appointmentv1.ListExaminationReportsResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	page, err := l.svcCtx.PatientManager.ListMyExaminationReports(l.ctx, principal, in.GetPage(), in.GetPageSize())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	response := &appointmentv1.ListExaminationReportsResponse{Page: page.Page, PageSize: page.PageSize, Total: page.Total, Reports: make([]*appointmentv1.ExaminationReport, 0, len(page.Items))}
	for _, value := range page.Items {
		response.Reports = append(response.Reports, reportResponse(value))
	}
	return response, nil
}
