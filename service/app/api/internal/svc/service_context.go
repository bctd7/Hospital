// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"hospital/service/app/api/internal/config"
	"hospital/service/identity/rpc/identityservice"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config   config.Config
	Identity identityservice.IdentityService
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		Identity: identityservice.NewIdentityService(zrpc.MustNewClient(c.IdentityRPC)),
	}
}
