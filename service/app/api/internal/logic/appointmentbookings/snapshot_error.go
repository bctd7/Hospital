package appointmentbookings

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func identitySnapshotNotFound() error {
	return status.Error(codes.FailedPrecondition, "organization display data is unavailable")
}
