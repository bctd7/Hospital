package logic

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/service/identity/rpc/internal/account"
	"hospital/service/identity/rpc/internal/authorization"
	"hospital/service/identity/rpc/internal/login"
)

func accountRPCError(err error) error {
	switch {
	case errors.Is(err, login.ErrInvalidCredential):
		return status.Error(codes.Unauthenticated, "wechat login credential is invalid or expired")
	case errors.Is(err, login.ErrProviderNotConfigured):
		return status.Error(codes.FailedPrecondition, "wechat login is not configured")
	case errors.Is(err, login.ErrProviderUnavailable):
		return status.Error(codes.Unavailable, "wechat login provider is unavailable")
	case errors.Is(err, account.ErrInvalidPhone), errors.Is(err, account.ErrInvalidExternalIdentity):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, account.ErrPhoneInUse), errors.Is(err, authorization.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, authorization.ErrNotFound):
		return status.Error(codes.NotFound, "identity account not found")
	case errors.Is(err, authorization.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	default:
		return status.Error(codes.Internal, "identity account operation failed")
	}
}
