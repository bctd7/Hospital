package logic

import (
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/session"
)

func tokenPairResponse(pair session.TokenPair) *identityv1.TokenPair {
	return &identityv1.TokenPair{
		AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken,
		AccessExpiresInSeconds:  remainingSeconds(pair.AccessExpiresAt),
		RefreshExpiresInSeconds: remainingSeconds(pair.RefreshExpiresAt),
	}
}

func remainingSeconds(expiresAt time.Time) int64 {
	seconds := int64(time.Until(expiresAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func sessionRPCError(err error) error {
	switch {
	case errors.Is(err, session.ErrInvalidRefreshToken),
		errors.Is(err, session.ErrSessionNotFound),
		errors.Is(err, session.ErrSessionExpired),
		errors.Is(err, session.ErrRefreshTokenReused),
		errors.Is(err, authn.ErrInactiveAccount):
		return status.Error(codes.Unauthenticated, "refresh token is invalid or expired")
	default:
		return status.Error(codes.Internal, "identity session operation failed")
	}
}
