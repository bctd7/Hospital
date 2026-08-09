package logging

import (
	"context"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
)

// Info records a successful or expected functional event.
func Info(ctx context.Context, event string, fields ...logx.LogField) {
	logc.Infow(ctx, event, prependEvent(event, fields)...)
}

// Error records a failed functional event without exposing it to the client response.
func Error(ctx context.Context, event string, err error, fields ...logx.LogField) {
	fields = prependEvent(event, fields)
	if err != nil {
		fields = append(fields, logx.Field(FieldError, err))
	}

	logc.Errorw(ctx, event, fields...)
}

func prependEvent(event string, fields []logx.LogField) []logx.LogField {
	result := make([]logx.LogField, 0, len(fields)+1)
	result = append(result, logx.Field(FieldEvent, event))
	return append(result, fields...)
}
