// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appointmentresources

import (
	"context"

	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListRoomExaminationItemsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListRoomExaminationItemsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListRoomExaminationItemsLogic {
	return &ListRoomExaminationItemsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListRoomExaminationItemsLogic) ListRoomExaminationItems(req *types.ListRoomExaminationItemsAPIRequest) (resp *types.ListRoomExaminationItemsAPIResponse, err error) {
	return listRoomItems(l.ctx, l.svcCtx, req)
}
