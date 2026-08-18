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

type GetExaminationItemConfigurationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetExaminationItemConfigurationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemConfigurationLogic {
	return &GetExaminationItemConfigurationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetExaminationItemConfigurationLogic) GetExaminationItemConfiguration(req *types.ExaminationItemConfigurationPathRequest) (resp *types.ExaminationItemConfigurationResponse, err error) {
	ctx, err := l.svcCtx.AuthenticatedRPCContext(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Guidance.GetExaminationItemConfiguration(ctx, &guidancev1.GetExaminationItemConfigurationRequest{
		ItemId: req.ItemID, RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return configurationResponse(value), nil
}
