package logic

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/catalog"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

func catalogPrincipal(ctx context.Context) (authn.Principal, error) {
	principal, err := authn.PrincipalFromContext(ctx)
	if err != nil {
		return authn.Principal{}, status.Error(codes.Unauthenticated, "authentication required")
	}
	return principal, nil
}

func catalogRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, catalog.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid examination catalog request")
	case errors.Is(err, catalog.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, catalog.ErrNotFound):
		return status.Error(codes.NotFound, "examination item not found")
	case errors.Is(err, catalog.ErrConflict):
		return status.Error(codes.AlreadyExists, "examination item conflict")
	case errors.Is(err, catalog.ErrVersionConflict):
		return status.Error(codes.Aborted, "examination item conflict")
	case errors.Is(err, catalog.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "examination item state does not allow the operation")
	case errors.Is(err, catalog.ErrNotImplemented):
		return status.Error(codes.Unimplemented, "examination catalog operation is not implemented")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func examinationItemResponse(item catalog.ExaminationItem) *appointmentv1.ExaminationItem {
	return &appointmentv1.ExaminationItem{
		ItemId:            item.ItemID,
		OwnerDepartmentId: item.OwnerDepartmentID,
		Name:              item.Name,
		Description:       item.Description,
		Status:            string(item.Status),
		Version:           item.Version,
		CreatedAt:         item.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:         item.UpdatedAt.UTC().Format(timeLayout),
	}
}

func changeStatusCommand(in *appointmentv1.ChangeExaminationItemStatusRequest) catalog.ChangeStatusCommand {
	return catalog.ChangeStatusCommand{
		ItemID:          in.ItemId,
		ExpectedVersion: in.ExpectedVersion,
		OperationID:     in.OperationId,
		RequestID:       in.RequestId,
	}
}
