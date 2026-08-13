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

type ListBookingOptionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListBookingOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBookingOptionsLogic {
	return &ListBookingOptionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListBookingOptionsLogic) ListBookingOptions(req *types.BookingOptionsPathRequest) (resp *types.ListBookingOptionsResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.ListBookingOptions(rpcCtx, &appointmentv1.ListBookingOptionsRequest{ItemId: req.ItemID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	response := &types.ListBookingOptionsResponse{WeekStartDate: value.WeekStartDate, WeekEndDate: value.WeekEndDate}
	response.Options = make([]types.BookingOptionResponse, 0, len(value.Options))
	for _, option := range value.Options {
		response.Options = append(response.Options, types.BookingOptionResponse{
			ItemID: option.ItemId, RoomID: option.RoomId, RoomDisplayName: option.RoomDisplayName,
			CampusID: option.CampusId, Building: option.Building, FloorNumber: option.FloorNumber,
			RoomNumber: option.RoomNumber, ServiceDate: option.ServiceDate, Session: option.Session,
			RoomOpenTime: option.RoomOpenTime, RoomCloseTime: option.RoomCloseTime,
			ItemStartTime: option.ItemStartTime, ItemEndTime: option.ItemEndTime,
			BookingCutoffTime: option.BookingCutoffTime, TotalCapacity: option.TotalCapacity,
			RemainingCapacity: option.RemainingCapacity,
		})
	}
	return response, nil
}
