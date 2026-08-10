package main

import (
	"flag"
	"fmt"

	"hospital/common/authn"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/server"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/identity-rpc.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}
	defer ctx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		identityv1.RegisterIdentityServiceServer(grpcServer, server.NewIdentityServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(authn.UnaryServerInterceptor(
		ctx.TokenManager,
		ctx.AuthorizationVersionValidator,
		identityv1.IdentityService_SendPhoneLoginCode_FullMethodName,
		identityv1.IdentityService_PhoneLogin_FullMethodName,
		identityv1.IdentityService_WeChatLogin_FullMethodName,
		identityv1.IdentityService_RefreshAccessToken_FullMethodName,
		identityv1.IdentityService_RevokeRefreshToken_FullMethodName,
		identityv1.IdentityService_GetOrganizationContext_FullMethodName,
		identityv1.IdentityService_ListDepartments_FullMethodName,
	))
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
