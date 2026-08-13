package logic

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authenticationmanager "hospital/service/identity/rpc/internal/authentication/manager"
	login "hospital/service/identity/rpc/internal/authentication/provider"
)

func accountRPCError(err error) error {
	switch {
	case errors.Is(err, login.ErrInvalidCredential):
		return status.Error(codes.Unauthenticated, "login credential is invalid or expired")
	case errors.Is(err, login.ErrProviderNotConfigured):
		return status.Error(codes.FailedPrecondition, "login provider is not configured")
	case errors.Is(err, login.ErrProviderUnavailable):
		return status.Error(codes.Unavailable, "login provider is unavailable")
	case errors.Is(err, login.ErrRateLimited):
		return status.Error(codes.ResourceExhausted, "verification requests are too frequent")
	case errors.Is(err, authenticationmanager.ErrInvalidPhone), errors.Is(err, authenticationmanager.ErrInvalidExternalIdentity):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, authenticationmanager.ErrPhoneInUse):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, authenticationmanager.ErrVerifiedPhoneChange):
		return status.Error(codes.FailedPrecondition, "verified login phone requires a dedicated change flow")
	default:
		return status.Error(codes.Internal, "identity account operation failed")
	}
}
