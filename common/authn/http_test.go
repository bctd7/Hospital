package authn

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPMiddlewareAddsPrincipalAndRawToken(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewTokenManager(TokenConfig{
		Issuer: "issuer", Audience: "audience", SigningKey: privateKey,
		VerificationKey: publicKey, TTL: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _, err := manager.Issue(Principal{AccountID: "account-1", Status: AccountStatusActive})
	if err != nil {
		t.Fatal(err)
	}
	handler := HTTPMiddleware(manager)(func(w http.ResponseWriter, r *http.Request) {
		principal, principalErr := PrincipalFromContext(r.Context())
		token, tokenErr := AccessTokenFromContext(r.Context())
		if principalErr != nil || tokenErr != nil || principal.AccountID != "account-1" || token != raw {
			t.Fatalf("authentication context was not populated")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}

func TestHTTPMiddlewareRejectsMissingToken(t *testing.T) {
	publicKey, _, _ := ed25519.GenerateKey(rand.Reader)
	manager, _ := NewTokenManager(TokenConfig{
		Issuer: "issuer", Audience: "audience", VerificationKey: publicKey, TTL: time.Minute,
	})
	response := httptest.NewRecorder()
	HTTPMiddleware(manager)(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler must not be called")
	})(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", response.Code)
	}
}
