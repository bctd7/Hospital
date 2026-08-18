// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package guidanceconfiguration

import (
	"context"

	"hospital/common/observability/logging"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfigureExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewConfigureExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfigureExaminationItemLogic {
	return &ConfigureExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConfigureExaminationItemLogic) ConfigureExaminationItem(req *types.ConfigureExaminationItemRequest) (resp *types.ExaminationItemConfigurationResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Guidance.ConfigureExaminationItem(ctx, &guidancev1.ConfigureExaminationItemRequest{
		Action: req.Action, ItemId: req.ItemID, OwnerDepartmentId: req.OwnerDepartmentID,
		ItemName: req.ItemName, EstimatedDurationMinutes: req.EstimatedDurationMinutes,
		ExpectedItemVersion: req.ExpectedItemVersion, Description: req.Description,
		PrecedenceRules: precedenceRequests(req.PrecedenceRules), PreparationRules: preparationRuleRequests(req.PreparationRules),
		Reminders: reminderRequests(req.Reminders), ExpectedConfigurationVersion: req.ExpectedConfigurationVersion,
		OperationId: req.OperationID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return configurationResponse(value), nil
}
