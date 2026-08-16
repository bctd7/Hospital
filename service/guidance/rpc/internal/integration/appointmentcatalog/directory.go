// Package appointmentcatalog 通过 Appointment RPC 读取 Guidance 所需的窄项目引用。
// 它不会访问 Appointment 数据库，也不会把项目配置复制成 Guidance 主数据。
package appointmentcatalog

import (
	"context"
	"fmt"

	"hospital/service/appointment/rpc/appointmentservice"
	"hospital/service/guidance/rpc/internal/rules/precedence"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Directory struct {
	client appointmentservice.AppointmentService
}

func New(client appointmentservice.AppointmentService) (*Directory, error) {
	if client == nil {
		return nil, fmt.Errorf("appointment project client is required")
	}
	return &Directory{client: client}, nil
}

func (d *Directory) ResolveProject(ctx context.Context, itemID string) (precedence.ProjectReference, error) {
	outgoing := ctx
	if incoming, ok := metadata.FromIncomingContext(ctx); ok {
		outgoing = metadata.NewOutgoingContext(ctx, incoming.Copy())
	}
	item, err := d.client.GetExaminationItemReference(outgoing, &appointmentservice.GetExaminationItemRequest{ItemId: itemID})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return precedence.ProjectReference{}, precedence.ErrInvalid
		case codes.NotFound:
			return precedence.ProjectReference{}, precedence.ErrNotFound
		case codes.Unauthenticated, codes.PermissionDenied:
			return precedence.ProjectReference{}, precedence.ErrForbidden
		default:
			return precedence.ProjectReference{}, fmt.Errorf("resolve appointment project: %w", err)
		}
	}
	return precedence.ProjectReference{
		ItemID:       item.ItemId,
		DepartmentID: item.OwnerDepartmentId,
		Name:         item.Name,
		Status:       item.Status,
		Version:      item.Version,
	}, nil
}

var _ precedence.ProjectDirectory = (*Directory)(nil)
