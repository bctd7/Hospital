package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListExaminationReportsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListExaminationReportsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListExaminationReportsLogic {
	return &ListExaminationReportsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListExaminationReportsLogic) ListExaminationReports(in *appointmentv1.ListExaminationReportsRequest) (*appointmentv1.ListExaminationReportsResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	page, err := l.svcCtx.StaffManager.ListExaminationReports(l.ctx, principal, in.GetDepartmentId(), in.GetKeyword(), in.GetPage(), in.GetPageSize())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	response := &appointmentv1.ListExaminationReportsResponse{
		Page: page.Page, PageSize: page.PageSize, Total: page.Total,
		Reports: make([]*appointmentv1.ExaminationReport, 0, len(page.Items)),
	}
	for _, value := range page.Items {
		response.Reports = append(response.Reports, reportResponse(value))
	}
	return response, nil
}
