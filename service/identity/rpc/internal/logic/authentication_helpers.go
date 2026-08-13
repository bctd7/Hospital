package logic

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/service/identity/rpc/internal/authentication"
	"hospital/service/identity/rpc/internal/authentication/sms"
)

func authenticationRPCError(err error) error {
	switch {
	case errors.Is(err, sms.ErrInvalidCode):
		return status.Error(codes.Unauthenticated, "login credential is invalid or expired")
	case errors.Is(err, sms.ErrNotConfigured):
		return status.Error(codes.FailedPrecondition, "SMS verification is not configured")
	case errors.Is(err, sms.ErrUnavailable):
		return status.Error(codes.Unavailable, "SMS verification is unavailable")
	case errors.Is(err, sms.ErrRateLimited):
		return status.Error(codes.ResourceExhausted, "verification requests are too frequent")
	case errors.Is(err, authentication.ErrInvalidPhone):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, "identity account operation failed")
	}
}
