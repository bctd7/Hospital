package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/catalog"
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
	principal, err := catalogPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	var itemInput *appointmentv1.ExaminationItemInput
	if in != nil {
		itemInput = in.ExaminationItem
	}
	if itemInput == nil {
		return nil, catalogRPCError(catalog.ErrInvalid)
	}
	item, err := l.svcCtx.CatalogManager.Create(l.ctx, principal, catalog.CreateCommand{
		OwnerDepartmentID: itemInput.OwnerDepartmentId,
		Name:              itemInput.Name,
		Description:       itemInput.Description,
		OperationID:       in.OperationId,
		RequestID:         in.RequestId,
	})
	if err != nil {
		return nil, catalogRPCError(err)
	}
	l.svcCtx.ResourceManager.InvalidateItem(l.ctx, item.OwnerDepartmentID, item.ItemID)
	return examinationItemResponse(item), nil
}
