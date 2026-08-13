// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"hospital/service/appointment/rpc/appointmentservice"
	"net/http"
	"time"

	"hospital/common/authn"
	"hospital/common/authn/versionredis"
	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/config"
	"hospital/service/app/api/internal/middleware"
	"hospital/service/identity/rpc/identityservice"

	redis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc/metadata"
)

type ServiceContext struct {
	Config      config.Config
	Identity    identityservice.IdentityService
	Appointment appointmentservice.AppointmentService
	AccessToken func(next http.HandlerFunc) http.HandlerFunc
	redisClient *redis.Client
}

func NewServiceContext(c config.Config) (*ServiceContext, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(c.Token.AccessPublicKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode app api access public key: %w", err)
	}
	tokenManager, err := authn.NewTokenManager(authn.TokenConfig{
		Issuer: c.Token.Issuer, Audience: c.Token.Audience,
		VerificationKey: ed25519.PublicKey(publicKeyBytes),
		TTL:             time.Duration(c.Token.AccessTTLSeconds) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create app api token verifier: %w", err)
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr: c.AuthorizationRedis.Addr, Password: c.AuthorizationRedis.Password, DB: c.AuthorizationRedis.DB,
	})
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("ping app api authorization redis: %w", err)
	}
	authorizationVersions, err := versionredis.NewStore(redisClient, c.AuthorizationRedis.Prefix)
	if err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("create app api authorization version store: %w", err)
	}
	authorizationVersionValidator, err := authn.NewAuthorizationVersionValidator(authorizationVersions)
	if err != nil {
		redisClient.Close()
		return nil, fmt.Errorf("create app api authorization version validator: %w", err)
	}
	accessToken := middleware.NewAccessTokenMiddleware(
		tokenManager,
		authorizationVersionValidator,
	)
	for _, method := range identityRPCMethodsWithSensitiveContent() {
		zrpc.DontLogClientContentForMethod(method)
	}

	return &ServiceContext{
		Config: c,
		Identity: identityservice.NewIdentityService(
			zrpc.MustNewClient(c.IdentityRPC),
		),
		Appointment: appointmentservice.NewAppointmentService(
			zrpc.MustNewClient(c.AppointmentRPC),
		),
		AccessToken: accessToken.Handle,
		redisClient: redisClient,
	}, nil
}

func identityRPCMethodsWithSensitiveContent() []string {
	return []string{
		identityv1.IdentityService_SendPhoneLoginCode_FullMethodName,
		identityv1.IdentityService_PhoneLogin_FullMethodName,
		identityv1.IdentityService_WeChatLogin_FullMethodName,
		identityv1.IdentityService_SetMyPhone_FullMethodName,
		identityv1.IdentityService_SearchAdminAccountByPhone_FullMethodName,
		identityv1.IdentityService_RefreshAccessToken_FullMethodName,
		identityv1.IdentityService_RevokeRefreshToken_FullMethodName,
	}
}

func (s *ServiceContext) Close() error {
	return s.redisClient.Close()
}

func (s *ServiceContext) AuthenticatedRPCContext(ctx context.Context) (context.Context, error) {
	raw, err := authn.AccessTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return metadata.AppendToOutgoingContext(
		ctx,
		"authorization", "Bearer "+raw,
		"x-request-id", logging.RequestIDFromContext(ctx),
	), nil
}
