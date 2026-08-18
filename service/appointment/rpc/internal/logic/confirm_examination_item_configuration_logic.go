package logic

import (
	"context"

	v1_appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConfirmExaminationItemConfigurationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConfirmExaminationItemConfigurationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConfirmExaminationItemConfigurationLogic {
	return &ConfirmExaminationItemConfigurationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ConfirmExaminationItemConfigurationLogic) ConfirmExaminationItemConfiguration(in *v1_appointmentv1.ExaminationItemConfigurationTransactionRequest) (*v1_appointmentv1.ExaminationItem, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.StaffManager.ConfirmProjectConfiguration(l.ctx, principal, in.GetTransactionId())
	if err != nil {
		return nil, projectRPCError(err)
	}
	return examinationItemResponse(item), nil
}
