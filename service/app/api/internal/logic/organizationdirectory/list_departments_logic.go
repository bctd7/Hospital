// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organizationdirectory

import (
	"context"

	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDepartmentsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListDepartmentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDepartmentsLogic {
	return &ListDepartmentsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDepartmentsLogic) ListDepartments(req *types.ListDepartmentsRequest) (resp *types.ListDepartmentsResponse, err error) {
	value, err := l.svcCtx.Identity.ListDepartments(l.ctx, &identityv1.ListDepartmentsRequest{
		CampusId:  req.CampusID,
		RequestId: logging.RequestIDFromContext(l.ctx),
	})
	if err != nil {
		return nil, err
	}
	return departmentsResponse(value.GetItems()), nil
}
