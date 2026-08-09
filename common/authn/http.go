package authn

import (
	"context"
	"net/http"
)

type accessTokenContextKey struct{}

func HTTPMiddleware(manager *TokenManager) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			raw, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			principal, err := manager.Verify(raw)
			if err != nil {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			ctx := ContextWithPrincipal(r.Context(), principal)
			ctx = context.WithValue(ctx, accessTokenContextKey{}, raw)
			next(w, r.WithContext(ctx))
		}
	}
}

func AccessTokenFromContext(ctx context.Context) (string, error) {
	token, ok := ctx.Value(accessTokenContextKey{}).(string)
	if !ok || token == "" {
		return "", ErrPrincipalMissing
	}
	return token, nil
}
