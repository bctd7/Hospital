package staff

import (
	"context"
	"testing"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
)

const bookingSummaryTestID = "40000000-0000-4000-8000-000000000007"

type bookingLookupStore struct {
	BookingStore
	value Booking
	err   error
}

func (s bookingLookupStore) GetBooking(context.Context, string) (Booking, error) {
	return s.value, s.err
}

type reportLookupStore struct {
	ReportStore
	value ExaminationReport
	err   error
}

func (s reportLookupStore) GetReportByBooking(context.Context, string, bool) (ExaminationReport, error) {
	return s.value, s.err
}

func bookingReadOperator() authn.Principal {
	return authn.Principal{
		AccountID:   "10000000-0000-4000-8000-000000000001",
		AccountType: authn.AccountTypeStaff,
		Status:      authn.AccountStatusActive,
		Roles:       []string{authn.RoleSuperAdmin},
		Permissions: []string{contractauthz.PermissionAppointmentRead},
	}
}

func TestGetBookingIncludesExistingReportSummary(t *testing.T) {
	manager := &Manager{
		bookings: bookingLookupStore{value: Booking{BookingID: bookingSummaryTestID, Status: BookingStatusInProgress}},
		reports: reportLookupStore{value: ExaminationReport{
			ReportID: "50000000-0000-4000-8000-000000000001",
			Status:   ReportStatusDraft,
			Version:  3,
		}},
	}

	value, err := manager.GetBooking(context.Background(), bookingReadOperator(), bookingSummaryTestID)
	if err != nil {
		t.Fatal(err)
	}
	if value.ReportID == "" || value.ReportStatus != ReportStatusDraft || value.ReportVersion != 3 {
		t.Fatalf("unexpected report summary: %#v", value)
	}
}

func TestGetBookingAllowsMissingReport(t *testing.T) {
	manager := &Manager{
		bookings: bookingLookupStore{value: Booking{BookingID: bookingSummaryTestID, Status: BookingStatusInProgress}},
		reports:  reportLookupStore{err: ErrNotFound},
	}

	value, err := manager.GetBooking(context.Background(), bookingReadOperator(), bookingSummaryTestID)
	if err != nil {
		t.Fatal(err)
	}
	if value.ReportID != "" || value.ReportStatus != "" || value.ReportVersion != 0 {
		t.Fatalf("missing report must stay empty: %#v", value)
	}
}
