package appointmentcatalog

import (
	"context"

	"hospital/common/observability/logging"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SaveExaminationItemReportTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSaveExaminationItemReportTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SaveExaminationItemReportTemplateLogic {
	return &SaveExaminationItemReportTemplateLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *SaveExaminationItemReportTemplateLogic) SaveExaminationItemReportTemplate(req *types.SaveExaminationItemReportTemplateRequest) (*types.ExaminationItemReportTemplateResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.SaveExaminationItemReportTemplate(ctx, &appointmentv1.SaveExaminationItemReportTemplateRequest{
		ItemId:                  req.ItemID,
		Content:                 &appointmentv1.ExaminationReportContent{ObjectiveFindings: req.ObjectiveFindings, Impression: req.Impression, Recommendation: req.Recommendation, Notes: req.Notes},
		ExpectedTemplateVersion: req.ExpectedTemplateVersion,
		OperationId:             req.OperationID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return examinationItemReportTemplateResponse(value), nil
}
