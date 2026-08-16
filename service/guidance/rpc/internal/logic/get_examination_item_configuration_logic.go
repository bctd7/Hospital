package logic

import (
	"context"

	v1_guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetExaminationItemConfigurationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExaminationItemConfigurationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemConfigurationLogic {
	return &GetExaminationItemConfigurationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetExaminationItemConfigurationLogic) GetExaminationItemConfiguration(in *v1_guidancev1.GetExaminationItemConfigurationRequest) (*v1_guidancev1.ExaminationItemConfiguration, error) {
	principal, err := guidancePrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if err := requireGuidanceStaff(principal); err != nil {
		return nil, err
	}
	result, err := l.svcCtx.ProjectConfigurationManager.Get(l.ctx, principal, in.GetItemId())
	if err != nil {
		return nil, projectConfigurationRPCError(err)
	}
	return configurationResponse(result), nil
}
