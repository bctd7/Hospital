package logic

import (
	"context"

	v1_appointmentv1 "hospital/contracts/gen/appointment/v1"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PrepareExaminationItemConfigurationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPrepareExaminationItemConfigurationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PrepareExaminationItemConfigurationLogic {
	return &PrepareExaminationItemConfigurationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Guidance 使用以下三个窄接口协调项目完整配置。预提交数据对普通项目查询不可见。
func (l *PrepareExaminationItemConfigurationLogic) PrepareExaminationItemConfiguration(in *v1_appointmentv1.PrepareExaminationItemConfigurationRequest) (*v1_appointmentv1.PreparedExaminationItemConfiguration, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	prepared, err := l.svcCtx.StaffManager.PrepareProjectConfiguration(l.ctx, principal, staffinput.PrepareProjectConfiguration{
		TransactionID: in.GetTransactionId(), Action: in.GetAction(), ItemID: in.GetItemId(),
		OwnerDepartmentID: in.GetOwnerDepartmentId(), Name: in.GetName(), Description: in.GetDescription(),
		EstimatedDurationMinutes: in.GetEstimatedDurationMinutes(), ExpectedVersion: in.GetExpectedVersion(),
		OperationID: in.GetOperationId(), RequestID: in.GetRequestId(),
	})
	if err != nil {
		return nil, projectRPCError(err)
	}
	return &v1_appointmentv1.PreparedExaminationItemConfiguration{
		TransactionId: prepared.TransactionID, Action: prepared.Action, ItemId: prepared.ItemID,
		ExpectedVersion: prepared.ExpectedVersion, State: prepared.State,
	}, nil
}
