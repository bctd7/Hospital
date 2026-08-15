package logic

import (
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/service/appointment/rpc/internal/manager/common"
)

func TestBookingRPCErrorIncludesPatientItemSessionReason(t *testing.T) {
	value, ok := status.FromError(bookingRPCError(common.ErrPatientItemSessionOccupied))
	if !ok {
		t.Fatal("bookingRPCError did not return a gRPC status")
	}
	if value.Code() != codes.FailedPrecondition {
		t.Fatalf("code = %v, want %v", value.Code(), codes.FailedPrecondition)
	}
	for _, detail := range value.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if ok && info.GetReason() == "PATIENT_ITEM_SESSION_OCCUPIED" && info.GetDomain() == "hospital.appointment" {
			return
		}
	}
	t.Fatal("missing PATIENT_ITEM_SESSION_OCCUPIED ErrorInfo")
}

func TestBookingRPCErrorIncludesWeeklyQuotaReason(t *testing.T) {
	value, ok := status.FromError(bookingRPCError(common.ErrPatientWeeklyQuotaFull))
	if !ok {
		t.Fatal("bookingRPCError did not return a gRPC status")
	}
	if value.Code() != codes.ResourceExhausted {
		t.Fatalf("code = %v, want %v", value.Code(), codes.ResourceExhausted)
	}
	for _, detail := range value.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if ok && info.GetReason() == "PATIENT_WEEKLY_QUOTA_EXHAUSTED" && info.GetDomain() == "hospital.appointment" {
			return
		}
	}
	t.Fatal("missing PATIENT_WEEKLY_QUOTA_EXHAUSTED ErrorInfo")
}
