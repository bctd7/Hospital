// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"

	"hospital/common/authn"
	"hospital/service/app/api/internal/svc"
	"hospital/service/app/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CurrentIdentityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCurrentIdentityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CurrentIdentityLogic {
	return &CurrentIdentityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CurrentIdentityLogic) CurrentIdentity() (resp *types.CurrentIdentityResponse, err error) {
	principal, err := authn.PrincipalFromContext(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.CurrentIdentityResponse{
		AccountID: principal.AccountID, AccountType: principal.AccountType, Status: principal.Status,
		Roles: principal.Roles, DepartmentID: principal.DepartmentID, Permissions: principal.Permissions,
		AuthorizationVersion: principal.AuthorizationVersion,
	}, nil
}
