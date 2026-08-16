package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveExaminationItemReportTemplateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSaveExaminationItemReportTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveExaminationItemReportTemplateLogic {
	return &SaveExaminationItemReportTemplateLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SaveExaminationItemReportTemplateLogic) SaveExaminationItemReportTemplate(in *appointmentv1.SaveExaminationItemReportTemplateRequest) (*appointmentv1.ExaminationItemReportTemplate, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.StaffManager.SaveProjectReportTemplate(l.ctx, principal, staffinput.SaveReportTemplate{
		ItemID: in.GetItemId(), Template: reportContent(in.GetContent()),
		ExpectedTemplateVersion: in.GetExpectedTemplateVersion(),
		Operation:               staffinput.Operation{OperationID: in.GetOperationId(), RequestID: in.GetRequestId()},
	})
	if err != nil {
		return nil, projectRPCError(err)
	}
	l.svcCtx.StaffManager.InvalidateItem(l.ctx, item.OwnerDepartmentID, item.ItemID)
	return reportTemplateResponse(item), nil
}

func reportTemplateResponse(item common.ExaminationItem) *appointmentv1.ExaminationItemReportTemplate {
	return &appointmentv1.ExaminationItemReportTemplate{
		ItemId: item.ItemID,
		Content: &appointmentv1.ExaminationReportContent{
			ObjectiveFindings: item.ReportTemplate.ObjectiveFindings,
			Impression:        item.ReportTemplate.Impression,
			Recommendation:    item.ReportTemplate.Recommendation,
			Notes:             item.ReportTemplate.Notes,
		},
		Version: item.ReportTemplate.Version, UpdatedAt: item.UpdatedAt.UTC().Format(timeLayout),
	}
}
