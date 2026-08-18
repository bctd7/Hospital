package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfigureExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfigureExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfigureExaminationItemLogic {
	return &ConfigureExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ConfigureExaminationItemLogic) ConfigureExaminationItem(in *v1_guidancev1.ConfigureExaminationItemRequest) (*v1_guidancev1.ExaminationItemConfiguration, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if err := requireGuidanceStaff(principal); err != nil {
		return nil, err
	}
	result, err := l.svcCtx.ProjectConfigurationManager.Configure(l.ctx, principal, configurationCommand(in))
	if err != nil {
		l.Errorf("configure examination item failed: %v", err)
		return nil, projectConfigurationRPCError(err)
	}
	return configurationResponse(result), nil
}
