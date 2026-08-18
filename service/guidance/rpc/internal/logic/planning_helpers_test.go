package logic

import (
	"testing"

	"hospital/service/guidance/rpc/internal/planning"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPlanningRPCErrorExplainsWeeklyQuotaExhausted(t *testing.T) {
	value := status.Convert(planningRPCError(planning.ErrWeeklyQuotaFull))
	if value.Code() != codes.FailedPrecondition {
		t.Fatalf("code = %s, want %s", value.Code(), codes.FailedPrecondition)
	}
	if value.Message() != "本周预约额度不足，无法一次性预约该方案中的全部检查项目" {
		t.Fatalf("message = %q", value.Message())
	}

	for _, detail := range value.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.GetReason() == "SMART_APPOINTMENT_WEEKLY_QUOTA_EXHAUSTED" {
			return
		}
	}
	t.Fatal("missing SMART_APPOINTMENT_WEEKLY_QUOTA_EXHAUSTED ErrorInfo")
}
