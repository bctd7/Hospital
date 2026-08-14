package staff

import (
	"errors"
	"testing"
	"time"
)

func TestValidateExaminationStartWindowUsesBookingSnapshot(t *testing.T) {
	serviceDate := time.Date(2026, 8, 15, 0, 0, 0, 0, bookingHospitalLocation)
	booking := Booking{
		ServiceDate:   serviceDate,
		ItemStartTime: "09:00:00",
		ItemEndTime:   "12:00:00",
	}

	for _, value := range []struct {
		name      string
		now       time.Time
		wantError bool
	}{
		{name: "before", now: time.Date(2026, 8, 15, 8, 59, 59, 0, bookingHospitalLocation), wantError: true},
		{name: "at start", now: time.Date(2026, 8, 15, 9, 0, 0, 0, bookingHospitalLocation)},
		{name: "inside", now: time.Date(2026, 8, 15, 11, 59, 59, 0, bookingHospitalLocation)},
		{name: "at end", now: time.Date(2026, 8, 15, 12, 0, 0, 0, bookingHospitalLocation), wantError: true},
		{name: "next day", now: time.Date(2026, 8, 16, 9, 0, 0, 0, bookingHospitalLocation), wantError: true},
	} {
		t.Run(value.name, func(t *testing.T) {
			err := validateExaminationStartWindow(booking, value.now)
			if value.wantError && !errors.Is(err, ErrExaminationWindowClosed) {
				t.Fatalf("error = %v, want ErrExaminationWindowClosed", err)
			}
			if !value.wantError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
