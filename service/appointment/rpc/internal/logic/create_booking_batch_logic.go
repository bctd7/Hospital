package logic

import (
	"context"

	v1_appointmentv1 "hospital/contracts/gen/appointment/v1"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateBookingBatchLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateBookingBatchLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBookingBatchLogic {
	return &CreateBookingBatchLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateBookingBatchLogic) CreateBookingBatch(in *v1_appointmentv1.CreateBookingBatchRequest) (*v1_appointmentv1.CreateBookingBatchResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	items := make([]patientmanager.CreateBookingCommand, 0, len(in.GetItems()))
	for _, value := range in.GetItems() {
		items = append(items, patientmanager.CreateBookingCommand{
			ItemID: value.GetItemId(), RoomID: value.GetRoomId(), ServiceDate: value.GetServiceDate(),
			Session: patientmanager.Session(value.GetSession()),
		})
	}
	bookings, err := l.svcCtx.PatientManager.CreateBookingBatch(l.ctx, principal, patientmanager.CreateBookingBatchCommand{
		Items: items, OperationID: in.GetOperationId(), RequestID: in.GetRequestId(),
		PatientDisplayName: in.GetPatientDisplayName(), PatientPhoneMasked: in.GetPatientPhoneMasked(),
	})
	if err != nil {
		return nil, bookingRPCError(err)
	}
	response := &v1_appointmentv1.CreateBookingBatchResponse{Bookings: make([]*v1_appointmentv1.Booking, 0, len(bookings))}
	for _, booking := range bookings {
		response.Bookings = append(response.Bookings, bookingResponse(booking))
	}
	return response, nil
}
