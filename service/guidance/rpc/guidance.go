package main

import (
	"flag"
	"fmt"

	"hospital/common/authn"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/config"
	"hospital/service/guidance/rpc/internal/server"
	"hospital/service/guidance/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/guidance.yaml", "the config file")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}
	defer svcCtx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		guidancev1.RegisterGuidanceServiceServer(grpcServer, server.NewGuidanceServiceServer(svcCtx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(authn.UnaryServerInterceptor(
		svcCtx.TokenManager,
		svcCtx.AuthorizationVersionValidator,
	))
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
