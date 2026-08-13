package logic

import (
	"context"

	"hospital/common/authn"
	"hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
	sharedmanager "hospital/service/appointment/rpc/internal/manager/shared"
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
		return nil, projectRPCError(common.ErrInvalid)
	}
	if in.Audience != "" && in.Audience != "patient" && in.Audience != "staff" {
		return nil, projectRPCError(common.ErrInvalid)
	}
	patientAudience := in.Audience == "patient" || (in.Audience == "" && principal.AccountType == authn.AccountTypePatient)
	if patientAudience {
		result, err := l.svcCtx.SharedManager.ListPatientProjects(l.ctx, principal, in.OwnerDepartmentId, in.Page, in.PageSize)
		if err != nil {
			return nil, projectRPCError(err)
		}
		items := make([]*appointmentv1.ExaminationItem, 0, len(result.Items))
		for _, item := range result.Items {
			items = append(items, examinationItemResponse(item))
		}
		return &appointmentv1.ListExaminationItemsResponse{Items: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total}, nil
	}
	result, err := l.svcCtx.SharedManager.ListStaffProjects(l.ctx, principal, sharedmanager.ListProjectsQuery{
		OwnerDepartmentID: in.OwnerDepartmentId,
		Status:            common.Status(in.Status),
		Page:              in.Page,
		PageSize:          in.PageSize,
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
