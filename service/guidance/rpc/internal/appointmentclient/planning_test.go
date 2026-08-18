package appointmentclient

import (
	"errors"
	"testing"

	"hospital/service/guidance/rpc/internal/planning"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapPlanningErrorRecognizesWeeklyQuotaExhausted(t *testing.T) {
	value, err := status.New(codes.ResourceExhausted, "weekly booking quota is exhausted").WithDetails(
		&errdetails.ErrorInfo{Reason: "PATIENT_WEEKLY_QUOTA_EXHAUSTED", Domain: "hospital.appointment"},
	)
	if err != nil {
		t.Fatalf("attach error details: %v", err)
	}

	got := mapPlanningError("create smart booking batch", value.Err())
	if !errors.Is(got, planning.ErrWeeklyQuotaFull) {
		t.Fatalf("mapPlanningError() = %v, want ErrWeeklyQuotaFull", got)
	}
}

func TestMapPlanningErrorKeepsUnknownResourceExhaustedAsConflict(t *testing.T) {
	got := mapPlanningError("create smart booking batch", status.Error(codes.ResourceExhausted, "capacity is exhausted"))
	if !errors.Is(got, planning.ErrConflict) {
		t.Fatalf("mapPlanningError() = %v, want ErrConflict", got)
	}
}
