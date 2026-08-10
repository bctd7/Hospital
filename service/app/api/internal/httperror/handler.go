package httperror

import (
	"context"
	"net/http"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Handler converts downstream gRPC errors into the stable HTTP contract used
// by the miniapp. Unknown internal errors never expose database or provider
// details.
func Handler(_ context.Context, err error) (int, any) {
	value, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError, Response{
			Code: "INTERNAL_ERROR", Message: "internal server error",
		}
	}

	reason := grpcReason(value)
	statusCode := grpcHTTPStatus(value.Code(), reason)
	message := value.Message()
	if value.Code() == codes.Internal || value.Code() == codes.Unknown {
		message = "internal server error"
	}
	return statusCode, Response{Code: reason, Message: message}
}

func grpcReason(value *status.Status) string {
	for _, detail := range value.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.GetReason() != "" {
			return info.GetReason()
		}
	}
	switch value.Code() {
	case codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case codes.Unauthenticated:
		return "UNAUTHENTICATED"
	case codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case codes.NotFound:
		return "NOT_FOUND"
	case codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case codes.Aborted:
		return "VERSION_CONFLICT"
	case codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case codes.ResourceExhausted:
		return "RATE_LIMITED"
	case codes.Unavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "INTERNAL_ERROR"
	}
}

func grpcHTTPStatus(code codes.Code, reason string) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists, codes.Aborted:
		return http.StatusConflict
	case codes.FailedPrecondition:
		if reason == "INVALID_ORGANIZATION_HIERARCHY" {
			return http.StatusUnprocessableEntity
		}
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
