package common

import (
	"testing"
	"time"
)

func TestPatientMessagesKeepOnlyRemindersThatActuallyBecameDue(t *testing.T) {
	location, _ := time.LoadLocation("Asia/Shanghai")
	serviceDate := time.Date(2026, 8, 17, 0, 0, 0, 0, location)
	startedAt := time.Date(2026, 8, 17, 10, 45, 0, 0, location)
	booking := Booking{
		BookingID: "booking-1", ServiceDate: serviceDate, BookingCutoffTime: "11:30:00",
		Status: BookingStatusInProgress, StartedAt: &startedAt, CreatedAt: serviceDate,
	}
	messages := PatientMessages([]Booking{booking}, nil, time.Date(2026, 8, 17, 12, 0, 0, 0, location))
	if hasMessage(messages, "booking:booking-1:patient:arrival-30m") {
		t.Fatal("30 minute reminder must not exist when examination started before its trigger")
	}
	if !hasMessage(messages, "booking:booking-1:patient:arrival-60m") {
		t.Fatal("60 minute reminder should remain because examination started after its trigger")
	}
}

func TestDepartmentMessagesKeepDueAndOverdueIndependently(t *testing.T) {
	location, _ := time.LoadLocation("Asia/Shanghai")
	serviceDate := time.Date(2026, 8, 17, 0, 0, 0, 0, location)
	startedAt := time.Date(2026, 8, 17, 10, 0, 0, 0, location)
	completedAt := time.Date(2026, 8, 17, 12, 30, 0, 0, location)
	booking := Booking{
		BookingID: "booking-2", ServiceDate: serviceDate, ItemEndTime: "12:00:00",
		Status: BookingStatusCompleted, StartedAt: &startedAt, CompletedAt: &completedAt, CreatedAt: serviceDate,
	}
	messages := DepartmentMessages([]Booking{booking}, time.Date(2026, 8, 17, 14, 0, 0, 0, location))
	if !hasMessage(messages, "booking:booking-2:department:report-due") {
		t.Fatal("report due reminder should exist because report was incomplete at window end")
	}
	if hasMessage(messages, "booking:booking-2:department:report-overdue") {
		t.Fatal("overdue reminder must not exist because report completed before the second trigger")
	}
}

func TestApplyMessageReadsIsAccountSpecificInput(t *testing.T) {
	readAt := time.Now()
	values := []Message{{MessageKey: "one"}, {MessageKey: "two"}}
	if unread := ApplyMessageReads(values, map[string]time.Time{"one": readAt}); unread != 1 {
		t.Fatalf("expected one unread message, got %d", unread)
	}
	if values[0].ReadAt == nil || values[1].ReadAt != nil {
		t.Fatal("read timestamps were not applied by key")
	}
}

func hasMessage(values []Message, key string) bool {
	for _, value := range values {
		if value.MessageKey == key {
			return true
		}
	}
	return false
}
