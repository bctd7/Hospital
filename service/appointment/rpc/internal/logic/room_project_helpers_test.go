package logic

import (
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"hospital/service/appointment/rpc/internal/manager/common"
)

func TestRoomProjectRPCErrorMarksWindowConflict(t *testing.T) {
	value := status.Convert(roomProjectRPCError(common.ErrWindowConflict))
	if value.Code() != codes.FailedPrecondition {
		t.Fatalf("code = %v, want %v", value.Code(), codes.FailedPrecondition)
	}
	for _, detail := range value.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.GetReason() == "ITEM_ROOM_WINDOW_CONFLICT" {
			return
		}
	}
	t.Fatal("missing ITEM_ROOM_WINDOW_CONFLICT ErrorInfo")
}
