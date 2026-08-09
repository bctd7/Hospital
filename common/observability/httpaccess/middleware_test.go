package httpaccess

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	projectlog "hospital/common/observability/logging"

	"github.com/zeromicro/go-zero/core/logx"
)

func TestMiddlewareGeneratesRequestIDAndLogsSafeFields(t *testing.T) {
	var logs bytes.Buffer
	restoreLogWriter(t, &logs)

	handler := Middleware()(func(writer http.ResponseWriter, request *http.Request) {
		if got := projectlog.RequestIDFromContext(request.Context()); got == "" {
			t.Fatal("request ID was not attached to the request context")
		}
		_, _ = io.Copy(io.Discard, request.Body)
		writer.WriteHeader(http.StatusCreated)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/plans?token=query-secret", strings.NewReader("body-secret"))
	request.Header.Set("Authorization", "Bearer header-secret")
	request.Header.Set("Cookie", "session=cookie-secret")
	recorder := httptest.NewRecorder()

	handler(recorder, request)

	requestID := recorder.Header().Get(projectlog.HeaderRequestID)
	if !projectlog.IsValidRequestID(requestID) {
		t.Fatalf("response request ID = %q, want a valid ID", requestID)
	}

	output := logs.String()
	for _, expected := range []string{
		projectlog.EventHTTPRequestCompleted,
		`"http_method":"POST"`,
		`"http_path":"/api/v1/plans"`,
		`"http_status":201`,
		`"request_id":"` + requestID + `"`,
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("log output does not contain %q: %s", expected, output)
		}
	}

	for _, secret := range []string{"query-secret", "body-secret", "header-secret", "cookie-secret"} {
		if strings.Contains(output, secret) {
			t.Errorf("log output contains sensitive value %q: %s", secret, output)
		}
	}
}

func TestMiddlewarePropagatesValidRequestID(t *testing.T) {
	var logs bytes.Buffer
	restoreLogWriter(t, &logs)

	handler := Middleware()(func(writer http.ResponseWriter, request *http.Request) {
		if got := projectlog.RequestIDFromContext(request.Context()); got != "client-request-123" {
			t.Fatalf("request context ID = %q, want %q", got, "client-request-123")
		}
		writer.WriteHeader(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.Header.Set(projectlog.HeaderRequestID, "client-request-123")
	recorder := httptest.NewRecorder()

	handler(recorder, request)

	if got := recorder.Header().Get(projectlog.HeaderRequestID); got != "client-request-123" {
		t.Fatalf("response request ID = %q, want %q", got, "client-request-123")
	}
}

func TestMiddlewareReplacesInvalidRequestIDAndLogsError(t *testing.T) {
	var logs bytes.Buffer
	restoreLogWriter(t, &logs)

	handler := Middleware()(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "bad request", http.StatusBadRequest)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/plans", nil)
	request.Header.Set(projectlog.HeaderRequestID, "invalid\nforged")
	recorder := httptest.NewRecorder()

	handler(recorder, request)

	requestID := recorder.Header().Get(projectlog.HeaderRequestID)
	if requestID == "invalid\nforged" || !projectlog.IsValidRequestID(requestID) {
		t.Fatalf("response request ID = %q, want a newly generated ID", requestID)
	}
	if !strings.Contains(logs.String(), `"level":"error"`) {
		t.Fatalf("error response was not logged at error level: %s", logs.String())
	}
}

func TestMiddlewareLogsPanicsAsInternalServerErrors(t *testing.T) {
	var logs bytes.Buffer
	restoreLogWriter(t, &logs)

	handler := Middleware()(func(http.ResponseWriter, *http.Request) {
		panic("test panic")
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil)
	recorder := httptest.NewRecorder()

	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("middleware swallowed the panic instead of delegating recovery to go-zero")
			}
		}()
		handler(recorder, request)
	}()

	output := logs.String()
	if !strings.Contains(output, `"http_status":500`) {
		t.Fatalf("panic was not logged as status 500: %s", output)
	}
	if !strings.Contains(output, `"level":"error"`) {
		t.Fatalf("panic was not logged at error level: %s", output)
	}
}

func restoreLogWriter(t *testing.T, destination io.Writer) {
	t.Helper()

	previousWriter := logx.Reset()
	logx.SetWriter(logx.NewWriter(destination))
	t.Cleanup(func() {
		logx.Reset()
		if previousWriter != nil {
			logx.SetWriter(previousWriter)
		}
	})
}

func TestRequestContextCanBeUsedByFunctionalLogs(t *testing.T) {
	var logs bytes.Buffer
	restoreLogWriter(t, &logs)

	ctx := projectlog.WithRequestID(context.Background(), "request-functional-1")
	projectlog.Info(ctx, "planning.plan.created", logx.Field("plan_id", "plan-1"))

	output := logs.String()
	for _, expected := range []string{
		`"event":"planning.plan.created"`,
		`"request_id":"request-functional-1"`,
		`"plan_id":"plan-1"`,
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("functional log output does not contain %q: %s", expected, output)
		}
	}
}
