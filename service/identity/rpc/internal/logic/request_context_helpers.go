package logic

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	"hospital/common/observability/logging"
)

func authorizationRequestContext(ctx context.Context, requestID string) context.Context {
	if logging.IsValidRequestID(requestID) {
		return logging.WithRequestID(ctx, requestID)
	}
	return ctx
}

func authorizationOperator(ctx context.Context) (authn.Principal, error) {
	principal, err := authn.PrincipalFromContext(ctx)
	if err != nil {
		return authn.Principal{}, status.Error(codes.Unauthenticated, "authentication required")
	}
	return principal, nil
}
