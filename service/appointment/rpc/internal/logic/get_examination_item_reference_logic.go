package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetExaminationItemReferenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetExaminationItemReferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetExaminationItemReferenceLogic {
	return &GetExaminationItemReferenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetExaminationItemReferenceLogic) GetExaminationItemReference(in *appointmentv1.GetExaminationItemRequest) (*appointmentv1.ExaminationItem, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	item, err := l.svcCtx.SharedManager.GetProjectReference(l.ctx, principal, in.GetItemId())
	if err != nil {
		return nil, projectRPCError(err)
	}
	return &appointmentv1.ExaminationItem{
		ItemId:                   item.ItemID,
		OwnerDepartmentId:        item.DepartmentID,
		Name:                     item.Name,
		EstimatedDurationMinutes: item.EstimatedDurationMinutes,
		Status:                   string(item.Status),
		Version:                  item.Version,
	}, nil
}
