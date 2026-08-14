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

type GetMyBookingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMyBookingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyBookingLogic {
	return &GetMyBookingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyBookingLogic) GetMyBooking(req *types.BookingPathRequest) (resp *types.BookingResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.GetMyBooking(rpcCtx, &appointmentv1.GetBookingRequest{BookingId: req.BookingID, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return bookingWithOrganization(rpcCtx, l.svcCtx, requestID, value), nil
}
