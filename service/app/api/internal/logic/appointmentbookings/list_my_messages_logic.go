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

type ListMyMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMyMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyMessagesLogic {
	return &ListMyMessagesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMyMessagesLogic) ListMyMessages(req *types.ListMyMessagesAPIRequest) (resp *types.ListMessagesAPIResponse, err error) {
	rpcCtx, requestID, err := bookingRPCContext(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.Appointment.ListMyMessages(rpcCtx, &appointmentv1.ListMyMessagesRequest{Page: req.Page, PageSize: req.PageSize, RequestId: requestID})
	if err != nil {
		return nil, err
	}
	return messageListWithOrganizations(rpcCtx, l.svcCtx, requestID, value), nil
}
