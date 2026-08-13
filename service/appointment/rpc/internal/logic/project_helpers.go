package logic

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/common/authn"
	appointmentv1 "hospital/contracts/gen/appointment/v1"
	"hospital/service/appointment/rpc/internal/manager/common"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

func appointmentPrincipal(ctx context.Context) (authn.Principal, error) {
	principal, err := authn.PrincipalFromContext(ctx)
	if err != nil {
		return authn.Principal{}, status.Error(codes.Unauthenticated, "authentication required")
	}
	return principal, nil
}

func projectRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, common.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid examination project request")
	case errors.Is(err, common.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, common.ErrNotFound):
		return status.Error(codes.NotFound, "examination item not found")
	case errors.Is(err, common.ErrConflict):
		return status.Error(codes.AlreadyExists, "examination item conflict")
	case errors.Is(err, common.ErrVersionConflict):
		return status.Error(codes.Aborted, "examination item conflict")
	case errors.Is(err, common.ErrInvalidState):
		return status.Error(codes.FailedPrecondition, "examination item state does not allow the operation")
	case errors.Is(err, common.ErrNotImplemented):
		return status.Error(codes.Unimplemented, "examination project operation is not implemented")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func examinationItemResponse(item common.ExaminationItem) *appointmentv1.ExaminationItem {
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

func changeStatusInput(in *appointmentv1.ChangeExaminationItemStatusRequest) staffinput.ChangeProjectStatus {
	return staffinput.ChangeProjectStatus{
		ItemID:          in.ItemId,
		ExpectedVersion: in.ExpectedVersion,
		OperationID:     in.OperationId,
		RequestID:       in.RequestId,
	}
}
