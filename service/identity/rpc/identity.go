package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"hospital/common/authn"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/messaging/outbox"
	"hospital/service/identity/rpc/internal/server"
	"hospital/service/identity/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
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
	svcCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}
	defer svcCtx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		identityv1.RegisterIdentityServiceServer(grpcServer, server.NewIdentityServiceServer(svcCtx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(authn.UnaryServerInterceptor(
		svcCtx.Security.Token,
		svcCtx.Security.AuthorizationVersion,
		identityv1.IdentityService_SendPhoneLoginCode_FullMethodName,
		identityv1.IdentityService_PhoneLogin_FullMethodName,
		identityv1.IdentityService_WeChatLogin_FullMethodName,
		identityv1.IdentityService_RefreshAccessToken_FullMethodName,
		identityv1.IdentityService_RevokeRefreshToken_FullMethodName,
		identityv1.IdentityService_GetOrganizationContext_FullMethodName,
		identityv1.IdentityService_ListDepartments_FullMethodName,
		identityv1.IdentityService_ListDoctorsByDepartment_FullMethodName,
	))
	defer s.Stop()
	if svcCtx.Workers.OutboxPublisher != nil {
		publisherCtx, stopPublisher :=
			context.WithCancel(context.Background())

		publisherDone := make(chan struct{})

		go func() {
			defer close(publisherDone)

			runOutboxPublisher(
				publisherCtx,
				svcCtx.Workers.OutboxPublisher,
				time.Second,
			)
		}()

		defer func() {
			stopPublisher()
			<-publisherDone
		}()
	}
	if svcCtx.Workers.AuthorizationVersionConsumer != nil {
		consumerCtx, stopConsumer := context.WithCancel(context.Background())
		consumerDone := make(chan struct{})

		go func() {
			defer close(consumerDone)
			svcCtx.Workers.AuthorizationVersionConsumer.Run(consumerCtx, time.Second)
		}()

		defer func() {
			stopConsumer()
			<-consumerDone
		}()
	}
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}

func runOutboxPublisher(
	ctx context.Context,
	publisher *outbox.Publisher,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := publisher.PublishBatch(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}

			logx.Errorf(
				"publish identity outbox batch: %v",
				err,
			)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
