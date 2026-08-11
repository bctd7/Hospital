package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"hospital/common/authn"
	"hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/config"
	"hospital/service/identity/rpc/internal/outbox"
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
		svcCtx.TokenManager,
		svcCtx.AuthorizationVersionValidator,
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
	if svcCtx.OutboxPublisher != nil {
		publisherCtx, stopPublisher :=
			context.WithCancel(context.Background())

		publisherDone := make(chan struct{})

		go func() {
			defer close(publisherDone)

			runOutboxPublisher(
				publisherCtx,
				svcCtx.OutboxPublisher,
				time.Second,
			)
		}()

		defer func() {
			stopPublisher()
			<-publisherDone
		}()
	}
	if svcCtx.AuthorizationVersionConsumer != nil && svcCtx.AuthorizationVersionProjector != nil {
		consumerCtx, stopConsumer := context.WithCancel(context.Background())
		consumerDone := make(chan struct{})

		go func() {
			defer close(consumerDone)
			runAuthorizationVersionConsumer(
				consumerCtx,
				svcCtx.AuthorizationVersionConsumer,
				svcCtx.AuthorizationVersionProjector,
				time.Second,
			)
		}()

		defer func() {
			stopConsumer()
			<-consumerDone
		}()
	}
	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}

type authorizationMessageConsumer interface {
	FetchMessage(ctx context.Context) (*kgo.Record, error)
	CommitMessage(ctx context.Context, record *kgo.Record) error
}

type authorizationMessageProjector interface {
	Apply(ctx context.Context, messageKey string, message []byte) (bool, error)
}

func runAuthorizationVersionConsumer(
	ctx context.Context,
	consumer authorizationMessageConsumer,
	projector authorizationMessageProjector,
	retryInterval time.Duration,
) {
	for {
		record, err := consumer.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logx.Errorf("fetch identity authorization event: %v", err)
			if !waitForRetry(ctx, retryInterval) {
				return
			}
			continue
		}

		for {
			_, err = projector.Apply(ctx, string(record.Key), record.Value)
			if err == nil {
				break
			}
			if ctx.Err() != nil {
				return
			}
			logx.Errorf(
				"project identity authorization event topic=%s partition=%d offset=%d: %v",
				record.Topic,
				record.Partition,
				record.Offset,
				err,
			)
			if !waitForRetry(ctx, retryInterval) {
				return
			}
		}

		for {
			if err := consumer.CommitMessage(ctx, record); err == nil {
				break
			} else {
				if ctx.Err() != nil {
					return
				}
				logx.Errorf(
					"commit identity authorization event topic=%s partition=%d offset=%d: %v",
					record.Topic,
					record.Partition,
					record.Offset,
					err,
				)
			}
			if !waitForRetry(ctx, retryInterval) {
				return
			}
		}
	}
}

func waitForRetry(ctx context.Context, interval time.Duration) bool {
	if interval <= 0 {
		interval = time.Second
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
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
