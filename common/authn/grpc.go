package authn

import (
	"context"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"hospital/common/observability/logging"
)

const (
	grpcAuthorizationMetadata = "authorization"
	grpcRequestIDMetadata     = "x-request-id"
	grpcHealthCheckMethod     = "/grpc.health.v1.Health/Check"
)

func UnaryServerInterceptor(manager *TokenManager, validator PrincipalValidator, publicMethods ...string) grpc.UnaryServerInterceptor {
	public := make(map[string]struct{}, len(publicMethods)+1)
	public[grpcHealthCheckMethod] = struct{}{}
	for _, method := range publicMethods {
		if method = strings.TrimSpace(method); method != "" {
			public[method] = struct{}{}
		}
	}
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if requestID := firstMetadataValue(ctx, grpcRequestIDMetadata); logging.IsValidRequestID(requestID) {
			ctx = logging.WithRequestID(ctx, requestID)
		}
		if _, ok := public[info.FullMethod]; ok {
			return handler(ctx, request)
		}
		raw, err := bearerToken(firstMetadataValue(ctx, grpcAuthorizationMetadata))
		if err != nil {
			logging.Error(ctx, "auth.token.rejected", err, logx.Field("rpc_method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		principal, err := manager.Verify(raw)
		if err != nil {
			logging.Error(ctx, "auth.token.rejected", err, logx.Field("rpc_method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		if validator == nil {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		if err := validator.ValidatePrincipal(ctx, principal); err != nil {
			logging.Error(ctx, "auth.authorization_version.rejected", err, logx.Field("rpc_method", info.FullMethod))
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		return handler(ContextWithPrincipal(ctx, principal), request)
	}
}

func firstMetadataValue(ctx context.Context, key string) string {
	values := metadata.ValueFromIncomingContext(ctx, key)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func bearerToken(value string) (string, error) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("bearer token is missing")
	}
	return parts[1], nil
}
