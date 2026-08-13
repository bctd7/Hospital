package logic

import (
	"context"

	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListExaminationItemsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListExaminationItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListExaminationItemsLogic {
	return &ListExaminationItemsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListExaminationItemsLogic) ListExaminationItems(in *appointmentv1.ListExaminationItemsRequest) (*appointmentv1.ListExaminationItemsResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	if in == nil {
		return nil, projectRPCError(manager.ErrInvalid)
	}
	result, err := l.svcCtx.StaffManager.ListProjects(l.ctx, principal, staffmanager.ListProjectsQuery{
		OwnerDepartmentID: in.OwnerDepartmentId,
		Status:            manager.Status(in.Status),
		Page:              in.Page,
		PageSize:          in.PageSize,
		RequestID:         in.RequestId,
	})
	if err != nil {
		return nil, projectRPCError(err)
	}
	items := make([]*appointmentv1.ExaminationItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, examinationItemResponse(item))
	}
	return &appointmentv1.ListExaminationItemsResponse{
		Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total,
	}, nil
}
