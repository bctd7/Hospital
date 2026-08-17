package appointmentclient

import (
	"context"
	"fmt"

	"hospital/service/appointment/rpc/appointmentservice"
	"hospital/service/guidance/rpc/internal/rules/precedence"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PrecedenceDirectory 为独立先后规则接口提供只读项目资料。
// 它不会复制项目，也不会直接访问 Appointment 数据库。
type PrecedenceDirectory struct {
	client appointmentservice.AppointmentService
}

func NewPrecedenceDirectory(client appointmentservice.AppointmentService) (*PrecedenceDirectory, error) {
	if client == nil {
		return nil, fmt.Errorf("appointment project client is required")
	}
	return &PrecedenceDirectory{client: client}, nil
}

func (d *PrecedenceDirectory) ResolveProject(ctx context.Context, itemID string) (precedence.ProjectReference, error) {
	item, err := d.client.GetExaminationItemReference(outgoingContext(ctx), &appointmentservice.GetExaminationItemRequest{ItemId: itemID})
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
		ItemID: item.ItemId, DepartmentID: item.OwnerDepartmentId, Name: item.Name,
		Status: item.Status, Version: item.Version,
	}, nil
}

var _ precedence.ProjectDirectory = (*PrecedenceDirectory)(nil)
