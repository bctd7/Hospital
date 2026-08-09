package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	requestIDBytes     = 16
	maxRequestIDLength = 64
)

var fallbackRequestIDSequence atomic.Uint64

type requestIDContextKey struct{}

// NewRequestID creates an opaque identifier suitable for correlating one inbound request.
func NewRequestID() string {
	var value [requestIDBytes]byte
	if _, err := rand.Read(value[:]); err == nil {
		return hex.EncodeToString(value[:])
	}

	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), fallbackRequestIDSequence.Add(1))
}

// IsValidRequestID reports whether an externally supplied request ID is safe to log and propagate.
func IsValidRequestID(requestID string) bool {
	if len(requestID) == 0 || len(requestID) > maxRequestIDLength {
		return false
	}

	for i := range len(requestID) {
		char := requestID[i]
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' || char == '.' {
			continue
		}

		return false
	}

	return true
}

// WithRequestID stores a request ID for application code and attaches it to all context-aware logs.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	ctx = context.WithValue(ctx, requestIDContextKey{}, requestID)
	return logx.ContextWithFields(ctx, logx.Field(FieldRequestID, requestID))
}

// RequestIDFromContext returns the request ID previously attached to ctx.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}
