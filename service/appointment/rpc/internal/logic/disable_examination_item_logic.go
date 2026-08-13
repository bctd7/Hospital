package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DisableExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDisableExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableExaminationItemLogic {
	return &DisableExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *DisableExaminationItemLogic) DisableExaminationItem(in *appointmentv1.ChangeExaminationItemStatusRequest) (*appointmentv1.ExaminationItem, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, projectRPCError(common.ErrInvalid)
	}
	item, err := l.svcCtx.StaffManager.DisableProject(l.ctx, principal, changeStatusInput(in))
	if err != nil {
		return nil, projectRPCError(err)
	}
	l.svcCtx.StaffManager.InvalidateItem(l.ctx, item.OwnerDepartmentID, item.ItemID)
	return examinationItemResponse(item), nil
}
