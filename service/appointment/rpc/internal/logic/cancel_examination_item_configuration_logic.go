package logic

import (
	"context"

	v1_appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelExaminationItemConfigurationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelExaminationItemConfigurationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelExaminationItemConfigurationLogic {
	return &CancelExaminationItemConfigurationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelExaminationItemConfigurationLogic) CancelExaminationItemConfiguration(in *v1_appointmentv1.ExaminationItemConfigurationTransactionRequest) (*v1_appointmentv1.CancelExaminationItemConfigurationResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	cancelled, err := l.svcCtx.StaffManager.CancelProjectConfiguration(l.ctx, principal, in.GetTransactionId())
	if err != nil {
		return nil, projectRPCError(err)
	}
	return &v1_appointmentv1.CancelExaminationItemConfigurationResponse{TransactionId: in.GetTransactionId(), Cancelled: cancelled}, nil
}
