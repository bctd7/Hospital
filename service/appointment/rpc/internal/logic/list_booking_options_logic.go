package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListBookingOptionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListBookingOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListBookingOptionsLogic {
	return &ListBookingOptionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListBookingOptionsLogic) ListBookingOptions(in *appointmentv1.ListBookingOptionsRequest) (*appointmentv1.ListBookingOptionsResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	values, weekStart, weekEnd, err := l.svcCtx.PatientManager.ListBookingOptions(l.ctx, principal, in.GetItemId())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	response := &appointmentv1.ListBookingOptionsResponse{
		WeekStartDate: weekStart.Format("2006-01-02"), WeekEndDate: weekEnd.Format("2006-01-02"),
		Options: make([]*appointmentv1.BookingOption, 0, len(values)),
	}
	for _, value := range values {
		response.Options = append(response.Options, &appointmentv1.BookingOption{
			ItemId: value.ItemID, RoomId: value.RoomID, RoomDisplayName: value.RoomDisplayName,
			CampusId: value.CampusID, Building: value.Building, FloorNumber: value.FloorNumber,
			RoomNumber: value.RoomNumber, ServiceDate: value.ServiceDate.Format("2006-01-02"),
			Session: string(value.Session), RoomOpenTime: value.RoomOpenTime, RoomCloseTime: value.RoomCloseTime,
			ItemStartTime: value.ItemStartTime, ItemEndTime: value.ItemEndTime,
			BookingCutoffTime: value.BookingCutoffTime, TotalCapacity: value.TotalCapacity,
			RemainingCapacity: value.RemainingCapacity,
		})
	}
	return response, nil
}
