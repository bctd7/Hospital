package login

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeChatClientExchangesLoginCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("appid") != "app-id" || r.URL.Query().Get("js_code") != "login-code" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"openid":"openid-1","session_key":"server-secret"}`))
	}))
	defer server.Close()

	client := NewWeChatClient("app-id", "app-secret", server.URL, server.Client())
	session, err := client.ExchangeLoginCode(context.Background(), "login-code")
	if err != nil {
		t.Fatal(err)
	}
	if session.OpenID != "openid-1" || session.SessionKey != "server-secret" {
		t.Fatalf("unexpected session: %#v", session)
	}
}

func TestWeChatClientDoesNotLeakSecretOnTransportError(t *testing.T) {
	client := NewWeChatClient("app-id", "do-not-leak-secret", "http://127.0.0.1:1", &http.Client{})
	_, err := client.ExchangeLoginCode(context.Background(), "do-not-leak-code")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("expected provider unavailable, got %v", err)
	}
	if strings.Contains(err.Error(), "do-not-leak-secret") || strings.Contains(err.Error(), "do-not-leak-code") {
		t.Fatalf("provider error leaked credentials: %v", err)
	}
}

func TestWeChatClientRejectsProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code"}`))
	}))
	defer server.Close()

	client := NewWeChatClient("app-id", "app-secret", server.URL, server.Client())
	if _, err := client.ExchangeLoginCode(context.Background(), "bad-code"); err != ErrInvalidCredential {
		t.Fatalf("expected invalid credential, got %v", err)
	}
}
