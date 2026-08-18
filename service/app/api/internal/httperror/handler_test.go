package httperror

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerMapsOrganizationErrorDetails(t *testing.T) {
	value, err := status.New(codes.FailedPrecondition, "department has active doctors").WithDetails(
		&errdetails.ErrorInfo{Reason: "DEPARTMENT_HAS_ACTIVE_DOCTORS", Domain: "hospital.identity"},
	)
	if err != nil {
		t.Fatal(err)
	}
	statusCode, body := Handler(context.Background(), value.Err())
	if statusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", statusCode)
	}
	response := body.(Response)
	if response.Code != "DEPARTMENT_HAS_ACTIVE_DOCTORS" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestHandlerDoesNotExposeInternalError(t *testing.T) {
	statusCode, body := Handler(context.Background(), errors.New("database password leaked"))
	if statusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", statusCode)
	}
	response := body.(Response)
	if response.Message != "internal server error" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestHandlerKeepsAuthenticationAndAuthorizationDistinct(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want int
	}{
		{name: "missing or invalid session", code: codes.Unauthenticated, want: http.StatusUnauthorized},
		{name: "department scope denied", code: codes.PermissionDenied, want: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			statusCode, _ := Handler(context.Background(), status.Error(test.code, test.name))
			if statusCode != test.want {
				t.Fatalf("status=%d, want %d", statusCode, test.want)
			}
		})
	}
}
