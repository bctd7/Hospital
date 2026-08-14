package appointmentcatalog

import (
	"context"

	"hospital/common/observability/logging"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetExaminationItemReportTemplateLogic 读取项目当前报告模板，不在网关层复制业务规则。
type GetExaminationItemReportTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetExaminationItemReportTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemReportTemplateLogic {
	return &GetExaminationItemReportTemplateLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetExaminationItemReportTemplateLogic) GetExaminationItemReportTemplate(req *types.ExaminationItemPathRequest) (*types.ExaminationItemReportTemplateResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.GetExaminationItemReportTemplate(ctx, &appointmentv1.GetExaminationItemRequest{
		ItemId: req.ItemID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return examinationItemReportTemplateResponse(value), nil
}

func examinationItemReportTemplateResponse(value *appointmentv1.ExaminationItemReportTemplate) *types.ExaminationItemReportTemplateResponse {
	content := value.GetContent()
	return &types.ExaminationItemReportTemplateResponse{
		ItemID: value.GetItemId(), ObjectiveFindings: content.GetObjectiveFindings(), Impression: content.GetImpression(),
		Recommendation: content.GetRecommendation(), Notes: content.GetNotes(), Version: value.GetVersion(), UpdatedAt: value.GetUpdatedAt(),
	}
}
