// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentbookings

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListExaminationReportVersionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListExaminationReportVersionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListExaminationReportVersionsLogic {
	return &ListExaminationReportVersionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListExaminationReportVersionsLogic) ListExaminationReportVersions(req *types.ReportPathRequest) (resp *types.ListReportVersionsAPIResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.ListExaminationReportVersions(rpcCtx, &appointmentv1.ListExaminationReportVersionsRequest{ReportId: req.ReportID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	response := &types.ListReportVersionsAPIResponse{Versions: make([]types.ReportVersionResponse, 0, len(value.Versions))}
	for _, current := range value.Versions {
		response.Versions = append(response.Versions, reportVersion(current))
	}
	return response, nil
}
