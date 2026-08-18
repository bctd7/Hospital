package guidanceconfiguration

import (
	"context"

	"hospital/common/observability/logging"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetExaminationItemPatientRemindersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetExaminationItemPatientRemindersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemPatientRemindersLogic {
	return &GetExaminationItemPatientRemindersLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetExaminationItemPatientRemindersLogic) GetExaminationItemPatientReminders(req *types.ExaminationItemConfigurationPathRequest) (*types.ExaminationItemPatientRemindersResponse, error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Guidance.GetExaminationItemPatientReminders(ctx, &guidancev1.GetExaminationItemPatientRemindersRequest{
		ItemId: req.ItemID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return &types.ExaminationItemPatientRemindersResponse{ItemID: value.GetItemId(), Reminders: reminderResponses(value.GetReminders())}, nil
}
