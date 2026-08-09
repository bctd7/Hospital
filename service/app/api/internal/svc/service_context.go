// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"hospital/common/authn"
	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/app/api/internal/config"
	"hospital/service/app/api/internal/middleware"
	"hospital/service/identity/rpc/identityservice"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc/metadata"
)

type ServiceContext struct {
	Config      config.Config
	Identity    identityservice.IdentityService
	AccessToken func(next http.HandlerFunc) http.HandlerFunc
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
	accessToken := middleware.NewAccessTokenMiddleware(tokenManager)
	for _, method := range []string{
		identityv1.IdentityService_SendPhoneLoginCode_FullMethodName,
		identityv1.IdentityService_PhoneLogin_FullMethodName,
		identityv1.IdentityService_WeChatLogin_FullMethodName,
		identityv1.IdentityService_SetMyPhone_FullMethodName,
		identityv1.IdentityService_FindAccountByPhone_FullMethodName,
		identityv1.IdentityService_RefreshAccessToken_FullMethodName,
		identityv1.IdentityService_RevokeRefreshToken_FullMethodName,
	} {
		zrpc.DontLogClientContentForMethod(method)
	}
	return &ServiceContext{
		Config:      c,
		Identity:    identityservice.NewIdentityService(zrpc.MustNewClient(c.IdentityRPC)),
		AccessToken: accessToken.Handle,
	}, nil
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
