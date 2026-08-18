package logic

import (
	"context"

	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// GetExaminationItemPatientRemindersLogic 向已登录患者公开项目配置中的普通患者提醒。
type GetExaminationItemPatientRemindersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExaminationItemPatientRemindersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemPatientRemindersLogic {
	return &GetExaminationItemPatientRemindersLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetExaminationItemPatientRemindersLogic) GetExaminationItemPatientReminders(in *guidancev1.GetExaminationItemPatientRemindersRequest) (*guidancev1.ExaminationItemPatientReminders, error) {
	if _, err := guidancePrincipal(l.ctx); err != nil {
		return nil, err
	}
	reminders, err := l.svcCtx.ProjectConfigurationManager.PatientReminders(l.ctx, in.GetItemId())
	if err != nil {
		return nil, projectConfigurationRPCError(err)
	}
	return &guidancev1.ExaminationItemPatientReminders{ItemId: in.GetItemId(), Reminders: reminderResponses(reminders)}, nil
}
