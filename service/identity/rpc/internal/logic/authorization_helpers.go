package logic

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	"hospital/common/observability/logging"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/authorization"
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

func authorizationContextResponse(principal authn.Principal) *identityv1.AuthorizationContext {
	return &identityv1.AuthorizationContext{
		AccountId:            principal.AccountID,
		AccountType:          principal.AccountType,
		Status:               principal.Status,
		Roles:                principal.Roles,
		DepartmentId:         principal.DepartmentID,
		Permissions:          principal.Permissions,
		AuthorizationVersion: principal.AuthorizationVersion,
	}
}

func authorizationRPCError(err error) error {
	switch {
	case errors.Is(err, authorization.ErrInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, authorization.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, authorization.ErrNotFound):
		return status.Error(codes.NotFound, "identity resource not found")
	default:
		return status.Error(codes.Internal, "identity authorization operation failed")
	}
}
