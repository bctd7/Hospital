package appointmentclient

import (
	"context"
	"fmt"

	"hospital/service/appointment/rpc/appointmentservice"
	"hospital/service/guidance/rpc/internal/projectconfiguration"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Configuration 负责完整项目配置流程所需的 Appointment 操作。
// Appointment 是项目基础事实的所有者，因此项目查询和 TCC Try/Confirm/Cancel
// 都必须经由这个适配器完成。
type Configuration struct {
	client appointmentservice.AppointmentService
}

func NewConfiguration(client appointmentservice.AppointmentService) (*Configuration, error) {
	if client == nil {
		return nil, fmt.Errorf("appointment client is required")
	}
	return &Configuration{client: client}, nil
}

func (g *Configuration) ResolveProject(ctx context.Context, itemID string) (projectconfiguration.Project, error) {
	item, err := g.client.GetExaminationItemReference(outgoingContext(ctx), &appointmentservice.GetExaminationItemRequest{ItemId: itemID})
	if err != nil {
		return projectconfiguration.Project{}, mapConfigurationError("resolve project", err)
	}
	return configurationProject(item), nil
}

func (g *Configuration) Prepare(ctx context.Context, transactionID string, command projectconfiguration.Command) error {
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
		return mapConfigurationError("prepare project configuration", err)
	}
	return nil
}

func (g *Configuration) Confirm(ctx context.Context, transactionID string) (projectconfiguration.Project, error) {
	item, err := g.client.ConfirmExaminationItemConfiguration(outgoingContext(ctx), &appointmentservice.ExaminationItemConfigurationTransactionRequest{
		TransactionId: transactionID,
	})
	if err != nil {
		return projectconfiguration.Project{}, mapConfigurationError("confirm project configuration", err)
	}
	return configurationProject(item), nil
}

func (g *Configuration) Cancel(ctx context.Context, transactionID string) error {
	_, err := g.client.CancelExaminationItemConfiguration(outgoingContext(ctx), &appointmentservice.ExaminationItemConfigurationTransactionRequest{
		TransactionId: transactionID,
	})
	if err != nil {
		return mapConfigurationError("cancel project configuration", err)
	}
	return nil
}

func configurationProject(item *appointmentservice.ExaminationItem) projectconfiguration.Project {
	return projectconfiguration.Project{
		ItemID: item.GetItemId(), OwnerDepartmentID: item.GetOwnerDepartmentId(), Name: item.GetName(),
		Status: item.GetStatus(), Version: item.GetVersion(), EstimatedDurationMinutes: item.GetEstimatedDurationMinutes(),
	}
}

func mapConfigurationError(action string, err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return projectconfiguration.ErrInvalid
	case codes.NotFound:
		return projectconfiguration.ErrNotFound
	case codes.Unauthenticated, codes.PermissionDenied:
		return projectconfiguration.ErrForbidden
	case codes.AlreadyExists, codes.FailedPrecondition:
		return projectconfiguration.ErrConflict
	case codes.Aborted:
		return projectconfiguration.ErrVersionConflict
	default:
		return fmt.Errorf("%s: %w", action, err)
	}
}

var _ projectconfiguration.AppointmentParticipant = (*Configuration)(nil)
var _ projectconfiguration.ProjectDirectory = (*Configuration)(nil)
