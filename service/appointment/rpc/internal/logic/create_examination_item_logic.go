package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
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
	var itemInput *appointmentv1.ExaminationItemInput
	if in != nil {
		itemInput = in.ExaminationItem
	}
	if itemInput == nil {
		return nil, projectRPCError(manager.ErrInvalid)
	}
	item, err := l.svcCtx.StaffManager.CreateProject(l.ctx, principal, staffmanager.CreateProjectCommand{
		OwnerDepartmentID: itemInput.OwnerDepartmentId,
		Name:              itemInput.Name,
		Description:       itemInput.Description,
		OperationID:       in.OperationId,
		RequestID:         in.RequestId,
	})
	if err != nil {
		return nil, projectRPCError(err)
	}
	l.svcCtx.StaffManager.InvalidateItem(l.ctx, item.OwnerDepartmentID, item.ItemID)
	return examinationItemResponse(item), nil
}
