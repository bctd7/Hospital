// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAvailableRoomsByExaminationItemLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAvailableRoomsByExaminationItemLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAvailableRoomsByExaminationItemLogic {
	return &ListAvailableRoomsByExaminationItemLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAvailableRoomsByExaminationItemLogic) ListAvailableRoomsByExaminationItem(req *types.ItemRoomsPathRequest) (resp *types.ListRoomExaminationItemsAPIResponse, err error) {
	return listItemRooms(l.ctx, l.svcCtx, req)
}
