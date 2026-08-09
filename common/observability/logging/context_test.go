package logging

import (
	"context"
	"strings"
	"testing"
)

func TestNewRequestID(t *testing.T) {
	first := NewRequestID()
	second := NewRequestID()

	if !IsValidRequestID(first) {
		t.Fatalf("NewRequestID() = %q, want a valid request ID", first)
	}
	if first == second {
		t.Fatalf("NewRequestID() returned duplicate IDs: %q", first)
	}
}

func TestIsValidRequestID(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		want      bool
	}{
		{name: "simple", requestID: "request-123", want: true},
		{name: "allowed separators", requestID: "web_01.trace-id", want: true},
		{name: "empty", requestID: "", want: false},
		{name: "too long", requestID: strings.Repeat("a", maxRequestIDLength+1), want: false},
		{name: "space", requestID: "request 123", want: false},
		{name: "newline", requestID: "request\nforged", want: false},
		{name: "unicode", requestID: "请求-123", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsValidRequestID(test.requestID); got != test.want {
				t.Fatalf("IsValidRequestID(%q) = %v, want %v", test.requestID, got, test.want)
			}
		})
	}
}

func TestRequestIDContext(t *testing.T) {
	ctx := WithRequestID(context.Background(), "request-123")

	if got := RequestIDFromContext(ctx); got != "request-123" {
		t.Fatalf("RequestIDFromContext() = %q, want %q", got, "request-123")
	}
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Fatalf("RequestIDFromContext(empty context) = %q, want empty", got)
	}
}
