package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListExaminationReportVersionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListExaminationReportVersionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListExaminationReportVersionsLogic {
	return &ListExaminationReportVersionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListExaminationReportVersionsLogic) ListExaminationReportVersions(in *appointmentv1.ListExaminationReportVersionsRequest) (*appointmentv1.ListExaminationReportVersionsResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	values, err := l.svcCtx.StaffManager.ListExaminationReportVersions(l.ctx, principal, in.GetReportId())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	response := &appointmentv1.ListExaminationReportVersionsResponse{Versions: make([]*appointmentv1.ExaminationReportVersion, 0, len(values))}
	for _, value := range values {
		response.Versions = append(response.Versions, reportVersionResponse(value))
	}
	return response, nil
}
