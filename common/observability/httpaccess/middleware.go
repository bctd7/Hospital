package httpaccess

import (
	"context"
	"net/http"
	"time"

	projectlog "hospital/common/observability/logging"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

// Middleware returns a service-wide, body-safe HTTP access logging middleware.
func Middleware() rest.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			startedAt := time.Now()
			requestID := request.Header.Get(projectlog.HeaderRequestID)
			if !projectlog.IsValidRequestID(requestID) {
				requestID = projectlog.NewRequestID()
			}

			ctx := projectlog.WithRequestID(request.Context(), requestID)
			request = request.WithContext(ctx)
			writer.Header().Set(projectlog.HeaderRequestID, requestID)

			trackedWriter := newResponseWriter(writer)
			defer func() {
				if recovered := recover(); recovered != nil {
					if !trackedWriter.wroteHeader {
						trackedWriter.statusCode = http.StatusInternalServerError
					}
					logRequestCompleted(ctx, request, trackedWriter.statusCode, startedAt, true)
					panic(recovered)
				}

				logRequestCompleted(ctx, request, trackedWriter.statusCode, startedAt, false)
			}()

			next(trackedWriter, request)
		}
	}
}

func logRequestCompleted(
	ctx context.Context,
	request *http.Request,
	statusCode int,
	startedAt time.Time,
	forceError bool,
) {
	fields := []logx.LogField{
		logx.Field(projectlog.FieldHTTPMethod, request.Method),
		logx.Field(projectlog.FieldHTTPPath, request.URL.Path),
		logx.Field(projectlog.FieldHTTPStatus, statusCode),
		logx.Field(projectlog.FieldDurationMS, time.Since(startedAt).Milliseconds()),
	}

	if forceError || statusCode >= http.StatusBadRequest {
		projectlog.Error(ctx, projectlog.EventHTTPRequestCompleted, nil, fields...)
	} else {
		projectlog.Info(ctx, projectlog.EventHTTPRequestCompleted, fields...)
	}
}
