package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/catalog"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateExaminationItemLogic {
	return &UpdateExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateExaminationItemLogic) UpdateExaminationItem(in *appointmentv1.UpdateExaminationItemRequest) (*appointmentv1.ExaminationItem, error) {
	principal, err := catalogPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, catalogRPCError(catalog.ErrInvalid)
	}
	item, err := l.svcCtx.CatalogManager.Update(l.ctx, principal, catalog.UpdateCommand{
		ItemID:          in.ItemId,
		Name:            in.Name,
		Description:     in.Description,
		ExpectedVersion: in.ExpectedVersion,
		OperationID:     in.OperationId,
		RequestID:       in.RequestId,
	})
	if err != nil {
		return nil, catalogRPCError(err)
	}
	l.svcCtx.ResourceManager.InvalidateItem(l.ctx, item.OwnerDepartmentID, item.ItemID)
	return examinationItemResponse(item), nil
}
