package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetExaminationItemReportTemplateLogic 只负责把管理端读取请求适配到现有项目管理能力。
type GetExaminationItemReportTemplateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExaminationItemReportTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemReportTemplateLogic {
	return &GetExaminationItemReportTemplateLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetExaminationItemReportTemplateLogic) GetExaminationItemReportTemplate(in *appointmentv1.GetExaminationItemRequest) (*appointmentv1.ExaminationItemReportTemplate, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.SharedManager.GetStaffProject(l.ctx, principal, in.GetItemId())
	if err != nil {
		return nil, projectRPCError(err)
	}
	return reportTemplateResponse(item), nil
}
