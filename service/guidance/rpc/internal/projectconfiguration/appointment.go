package projectconfiguration

import (
	"context"
	"fmt"

	"hospital/service/appointment/rpc/appointmentservice"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AppointmentGateway 是 Guidance 访问 Appointment 的唯一窄适配器。
// 它只负责项目事实查询和配置事务的 Try/Confirm/Cancel，不持有 Appointment 数据。
type AppointmentGateway struct {
	client appointmentservice.AppointmentService
}

func NewAppointmentGateway(client appointmentservice.AppointmentService) (*AppointmentGateway, error) {
	if client == nil {
		return nil, fmt.Errorf("appointment client is required")
	}
	return &AppointmentGateway{client: client}, nil
}

func outgoingContext(ctx context.Context) context.Context {
	if incoming, ok := metadata.FromIncomingContext(ctx); ok {
		return metadata.NewOutgoingContext(ctx, incoming.Copy())
	}
	return ctx
}

func (g *AppointmentGateway) ResolveProject(ctx context.Context, itemID string) (Project, error) {
	item, err := g.client.GetExaminationItemReference(outgoingContext(ctx), &appointmentservice.GetExaminationItemRequest{ItemId: itemID})
	if err != nil {
		return Project{}, mapAppointmentError("resolve project", err)
	}
	return projectFromRPC(item), nil
}

func (g *AppointmentGateway) Prepare(ctx context.Context, transactionID string, command Command) error {
	_, err := g.client.PrepareExaminationItemConfiguration(outgoingContext(ctx), &appointmentservice.PrepareExaminationItemConfigurationRequest{
		TransactionId:            transactionID,
		Action:                   command.Action,
		ItemId:                   command.ItemID,
		OwnerDepartmentId:        command.OwnerDepartmentID,
		Name:                     command.ItemName,
		Description:              command.Description,
		EstimatedDurationMinutes: command.EstimatedDurationMinutes,
		ExpectedVersion:          command.ExpectedItemVersion,
		OperationId:              command.OperationID,
		RequestId:                command.RequestID,
	})
	if err != nil {
		return mapAppointmentError("prepare project configuration", err)
	}
	return nil
}

func (g *AppointmentGateway) Confirm(ctx context.Context, transactionID string) (Project, error) {
	item, err := g.client.ConfirmExaminationItemConfiguration(outgoingContext(ctx), &appointmentservice.ExaminationItemConfigurationTransactionRequest{
		TransactionId: transactionID,
	})
	if err != nil {
		return Project{}, mapAppointmentError("confirm project configuration", err)
	}
	return projectFromRPC(item), nil
}

func (g *AppointmentGateway) Cancel(ctx context.Context, transactionID string) error {
	_, err := g.client.CancelExaminationItemConfiguration(outgoingContext(ctx), &appointmentservice.ExaminationItemConfigurationTransactionRequest{
		TransactionId: transactionID,
	})
	if err != nil {
		return mapAppointmentError("cancel project configuration", err)
	}
	return nil
}

func projectFromRPC(item *appointmentservice.ExaminationItem) Project {
	return Project{
		ItemID: item.GetItemId(), OwnerDepartmentID: item.GetOwnerDepartmentId(), Name: item.GetName(),
		Status: item.GetStatus(), Version: item.GetVersion(), EstimatedDurationMinutes: item.GetEstimatedDurationMinutes(),
	}
}

func mapAppointmentError(action string, err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return ErrInvalid
	case codes.NotFound:
		return ErrNotFound
	case codes.Unauthenticated, codes.PermissionDenied:
		return ErrForbidden
	case codes.AlreadyExists, codes.FailedPrecondition:
		return ErrConflict
	case codes.Aborted:
		return ErrVersionConflict
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

var _ AppointmentParticipant = (*AppointmentGateway)(nil)
var _ ProjectDirectory = (*AppointmentGateway)(nil)
