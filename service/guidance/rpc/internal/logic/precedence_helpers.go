package logic

import (
	"context"
	"errors"
	"time"

	"hospital/common/authn"
	guidancev1 "hospital/contracts/gen/guidance/v1"
	"hospital/service/guidance/rpc/internal/rules/precedence"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const guidanceTimeLayout = "2006-01-02T15:04:05.000Z07:00"

func guidancePrincipal(ctx context.Context) (authn.Principal, error) {
	principal, err := authn.PrincipalFromContext(ctx)
	if err != nil {
		return authn.Principal{}, status.Error(codes.Unauthenticated, "authentication required")
	}
	return principal, nil
}

func precedenceRPCError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, precedence.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid precedence rule request")
	case errors.Is(err, precedence.ErrForbidden):
		return status.Error(codes.PermissionDenied, "permission denied")
	case errors.Is(err, precedence.ErrNotFound):
		return status.Error(codes.NotFound, "precedence rule or examination item not found")
	case errors.Is(err, precedence.ErrConflict):
		return status.Error(codes.AlreadyExists, "precedence rule already exists")
	case errors.Is(err, precedence.ErrCycle):
		return status.Error(codes.FailedPrecondition, "precedence rule would create a circular dependency")
	case errors.Is(err, precedence.ErrVersionConflict):
		return status.Error(codes.Aborted, "precedence rule version conflict")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func precedenceRuleResponse(relation precedence.Relation) *guidancev1.PrecedenceRule {
	return &guidancev1.PrecedenceRule{
		RuleId:                  relation.RuleID,
		PredecessorItemId:       relation.PredecessorItemID,
		PredecessorDepartmentId: relation.PredecessorDepartmentID,
		PredecessorItemName:     relation.PredecessorItemName,
		SuccessorItemId:         relation.SuccessorItemID,
		SuccessorDepartmentId:   relation.SuccessorDepartmentID,
		SuccessorItemName:       relation.SuccessorItemName,
		StaffReason:             relation.StaffReason,
		PatientMessage:          relation.PatientMessage,
		Direct:                  relation.Direct,
		PathLength:              int32(relation.PathLength),
		Version:                 relation.Version,
		CreatedBy:               relation.CreatedBy,
		CreatedAt:               formatGuidanceTime(relation.CreatedAt),
		UpdatedAt:               formatGuidanceTime(relation.UpdatedAt),
	}
}

func directRuleResponse(rule precedence.Rule) *guidancev1.PrecedenceRule {
	return precedenceRuleResponse(precedence.Relation{Rule: rule, Direct: true, PathLength: 1})
}

func formatGuidanceTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(guidanceTimeLayout)
}
