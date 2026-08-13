package logic

import (
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/service/appointment/rpc/internal/manager/common"
)

func TestBookingRPCErrorIncludesPatientSessionReason(t *testing.T) {
	value, ok := status.FromError(bookingRPCError(common.ErrPatientSessionOccupied))
	if !ok {
		t.Fatal("bookingRPCError did not return a gRPC status")
	}
	if value.Code() != codes.FailedPrecondition {
		t.Fatalf("code = %v, want %v", value.Code(), codes.FailedPrecondition)
	}
	for _, detail := range value.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if ok && info.GetReason() == "PATIENT_SESSION_OCCUPIED" && info.GetDomain() == "hospital.appointment" {
			return
		}
	}
	t.Fatal("missing PATIENT_SESSION_OCCUPIED ErrorInfo")
}
