package logic

import (
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/service/identity/rpc/internal/organization"
)

func TestOrganizationRPCErrorIncludesStableReason(t *testing.T) {
	err := organizationRPCError(organization.ErrActiveChildren)
	value := status.Convert(err)
	if value.Code() != codes.FailedPrecondition {
		t.Fatalf("unexpected code: %v", value.Code())
	}
	for _, detail := range value.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			if info.GetReason() != "ORGANIZATION_HAS_ACTIVE_CHILDREN" {
				t.Fatalf("unexpected reason: %q", info.GetReason())
			}
			return
		}
	}
	t.Fatal("missing ErrorInfo detail")
}
