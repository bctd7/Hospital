package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/catalog"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnableExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnableExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnableExaminationItemLogic {
	return &EnableExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EnableExaminationItemLogic) EnableExaminationItem(in *appointmentv1.ChangeExaminationItemStatusRequest) (*appointmentv1.ExaminationItem, error) {
	principal, err := catalogPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, catalogRPCError(catalog.ErrInvalid)
	}
	item, err := l.svcCtx.CatalogManager.Enable(l.ctx, principal, changeStatusCommand(in))
	if err != nil {
		return nil, catalogRPCError(err)
	}
	l.svcCtx.ResourceManager.InvalidateItem(l.ctx, item.OwnerDepartmentID, item.ItemID)
	return examinationItemResponse(item), nil
}
