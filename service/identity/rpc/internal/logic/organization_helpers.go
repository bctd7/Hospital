package logic

import (
	"errors"
	identityv1 "hospital/contracts/gen/identity/v1"
	"hospital/service/identity/rpc/internal/organization"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func organizationRPCError(err error) error {
	switch {
	case errors.Is(err, organization.ErrInvalid):
		return organizationStatusError(codes.InvalidArgument, err.Error(), "INVALID_ORGANIZATION_REQUEST")

	case errors.Is(err, organization.ErrForbidden):
		return organizationStatusError(codes.PermissionDenied, "permission denied", "PERMISSION_DENIED")

	case errors.Is(err, organization.ErrNotFound):
		return organizationStatusError(codes.NotFound, "organization unit not found", "ORGANIZATION_UNIT_NOT_FOUND")

	case errors.Is(err, organization.ErrConflict):
		return organizationStatusError(codes.AlreadyExists, err.Error(), "OPERATION_ID_REUSED")

	case errors.Is(err, organization.ErrVersionConflict):
		return organizationStatusError(codes.Aborted, "organization unit version conflict", "VERSION_CONFLICT")

	case errors.Is(err, organization.ErrInvalidHierarchy):
		return organizationStatusError(codes.FailedPrecondition, err.Error(), "INVALID_ORGANIZATION_HIERARCHY")

	case errors.Is(err, organization.ErrActiveChildren):
		return organizationStatusError(codes.FailedPrecondition, err.Error(), "ORGANIZATION_HAS_ACTIVE_CHILDREN")

	case errors.Is(err, organization.ErrActiveDoctors):
		return organizationStatusError(codes.FailedPrecondition, err.Error(), "DEPARTMENT_HAS_ACTIVE_DOCTORS")

	case errors.Is(err, organization.ErrDirectoryUnavailable):
		return organizationStatusError(codes.Internal, "organization directory is unavailable", "ORGANIZATION_DIRECTORY_UNAVAILABLE")

	default:
		return organizationStatusError(codes.Internal, "organization operation failed", "ORGANIZATION_OPERATION_FAILED")
	}
}

func organizationStatusError(code codes.Code, message, reason string) error {
	value := status.New(code, message)
	withDetails, err := value.WithDetails(&errdetails.ErrorInfo{
		Reason: reason,
		Domain: "hospital.identity",
	})
	if err != nil {
		return value.Err()
	}
	return withDetails.Err()
}

func adminOrganizationUnitResponse(unit organization.Unit) *identityv1.AdminOrganizationUnit {
	return &identityv1.AdminOrganizationUnit{
		UnitId:      unit.ID,
		ParentId:    unit.ParentID,
		UnitType:    string(unit.Type),
		Code:        unit.Code,
		Name:        unit.Name,
		Status:      string(unit.Status),
		ChildCount:  unit.ChildCount,
		DoctorCount: unit.DoctorCount,
		Version:     unit.Version,
	}
}

func adminOrganizationUnitsResponse(units []organization.Unit) *identityv1.ListOrganizationUnitsResponse {
	items := make([]*identityv1.AdminOrganizationUnit, 0, len(units))
	for _, unit := range units {
		items = append(items, adminOrganizationUnitResponse(unit))
	}
	return &identityv1.ListOrganizationUnitsResponse{Items: items}
}

func organizationContextResponse(value organization.DirectoryContext) *identityv1.OrganizationContext {
	campuses := make([]*identityv1.CampusSummary, 0, len(value.Campuses))
	for _, campus := range value.Campuses {
		campuses = append(campuses, &identityv1.CampusSummary{
			CampusId:        campus.ID,
			HospitalId:      campus.ParentID,
			Code:            campus.Code,
			Name:            campus.Name,
			DepartmentCount: campus.ChildCount,
			Status:          string(campus.Status),
			Version:         campus.Version,
		})
	}
	return &identityv1.OrganizationContext{
		Hospital: &identityv1.HospitalSummary{
			HospitalId: value.Hospital.ID,
			Code:       value.Hospital.Code,
			Name:       value.Hospital.Name,
			Version:    value.Hospital.Version,
		},
		Campuses: campuses,
	}
}

func directoryDepartmentsResponse(values []organization.DirectoryDepartment) *identityv1.ListDepartmentsResponse {
	items := make([]*identityv1.DepartmentSummary, 0, len(values))
	for _, value := range values {
		items = append(items, &identityv1.DepartmentSummary{
			DepartmentId: value.ID,
			CampusId:     value.ParentID,
			CampusName:   value.CampusName,
			Code:         value.Code,
			Name:         value.Name,
			DoctorCount:  value.DoctorCount,
			Status:       string(value.Status),
			Version:      value.Version,
		})
	}
	return &identityv1.ListDepartmentsResponse{Items: items}
}
