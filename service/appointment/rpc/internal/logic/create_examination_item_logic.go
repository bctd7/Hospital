package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateExaminationItemLogic {
	return &CreateExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateExaminationItemLogic) CreateExaminationItem(in *appointmentv1.CreateExaminationItemRequest) (*appointmentv1.ExaminationItem, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	itemInput := in.GetExaminationItem()
	if itemInput == nil {
		return nil, projectRPCError(common.ErrInvalid)
	}
	item, err := l.svcCtx.StaffManager.CreateProject(l.ctx, principal, staffinput.CreateProject{
		OwnerDepartmentID:        itemInput.OwnerDepartmentId,
		Name:                     itemInput.Name,
		Description:              itemInput.Description,
		EstimatedDurationMinutes: itemInput.EstimatedDurationMinutes,
		OperationID:              in.GetOperationId(),
		RequestID:                in.GetRequestId(),
	})
	if err != nil {
		return nil, projectRPCError(err)
	}
	l.svcCtx.StaffManager.InvalidateItem(l.ctx, item.OwnerDepartmentID, item.ItemID)
	return examinationItemResponse(item), nil
}
