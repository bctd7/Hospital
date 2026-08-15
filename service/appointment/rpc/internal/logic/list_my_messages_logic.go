package logic

import (
	"context"

	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMyMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyMessagesLogic {
	return &ListMyMessagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListMyMessagesLogic) ListMyMessages(in *appointmentv1.ListMyMessagesRequest) (*appointmentv1.ListMessagesResponse, error) {
	principal, err := appointmentPrincipal(l.ctx)
	if err != nil {
		return nil, err
	}
	page, err := l.svcCtx.PatientManager.ListMyMessages(l.ctx, principal, in.GetPage(), in.GetPageSize())
	if err != nil {
		return nil, bookingRPCError(err)
	}
	return messageListResponse(page), nil
}
