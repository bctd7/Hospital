package appointmentclient

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// outgoingContext 将入口 RPC 中的身份元数据继续传给 Appointment。
func outgoingContext(ctx context.Context) context.Context {
	if incoming, ok := metadata.FromIncomingContext(ctx); ok {
		return metadata.NewOutgoingContext(ctx, incoming.Copy())
	}
	return ctx
}
