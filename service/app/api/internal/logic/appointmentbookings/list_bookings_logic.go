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

type ListBookingsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListBookingsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBookingsLogic {
	return &ListBookingsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListBookingsLogic) ListBookings(req *types.ListBookingsAPIRequest) (resp *types.ListBookingsAPIResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.ListBookings(rpcCtx, &appointmentv1.ListBookingsRequest{
		DepartmentId: req.DepartmentID, ServiceDate: req.ServiceDate, Session: req.Session,
		ItemId: req.ItemID, RoomId: req.RoomID, Status: req.Status,
		PatientKeyword: req.PatientKeyword,
		View:           req.View,
		Page:           req.Page, PageSize: req.PageSize, RequestId: requestID,
	})
	if err != nil {
		return nil, err
	}
	return bookingListWithOrganizations(rpcCtx, l.svcCtx, requestID, value), nil
}
