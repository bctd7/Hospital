package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetExaminationItemLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemLogic {
	return &GetExaminationItemLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetExaminationItemLogic) GetExaminationItem(in *appointmentv1.GetExaminationItemRequest) (*appointmentv1.ExaminationItem, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, projectRPCError(common.ErrInvalid)
	}
	item, err := l.svcCtx.SharedManager.GetStaffProject(l.ctx, principal, in.ItemId)
	if err != nil {
		return nil, projectRPCError(err)
	}
	return examinationItemResponse(item), nil
}
