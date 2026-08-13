package mysqlstore

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"hospital/common/authn"
	appointmentmanager "hospital/service/appointment/rpc/internal/manager"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
)

const (
	bookingTestDepartment = "41000000-0000-0000-0000-000000000001"
	bookingTestCampus     = "41000000-0000-0000-0000-000000000002"
	bookingTestItem       = "41000000-0000-0000-0000-000000000003"
	bookingTestRoom       = "41000000-0000-0000-0000-000000000004"
	bookingTestRelation   = "41000000-0000-0000-0000-000000000005"
	bookingTestRoomWindow = "41000000-0000-0000-0000-000000000006"
	bookingTestItemWindow = "41000000-0000-0000-0000-000000000007"
	bookingTestPatient    = "41000000-0000-0000-0000-000000000008"
)

func TestBookingCapacityAllowsOnlyOneConcurrentWinner(t *testing.T) {
	dataSource := os.Getenv("APPOINTMENT_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("APPOINTMENT_TEST_MYSQL_DSN is not set")
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(location)
	if now.Hour() == 23 && now.Minute() >= 58 {
		t.Skip("not enough time remains in the service date for the booking cutoff")
	}
	serviceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekday := (int(serviceDate.Weekday())+6)%7 + 1

	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	cleanupBookingIntegrationData(t, store, ctx)
	defer cleanupBookingIntegrationData(t, store, ctx)
	seedBookingIntegrationData(t, store, ctx, weekday)

	manager, err := patientmanager.NewManager(store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	patient := authn.Principal{AccountID: bookingTestPatient, AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive}
	options, _, _, err := manager.ListBookingOptions(ctx, patient, bookingTestItem)
	if err != nil {
		t.Fatalf("list booking options: %v", err)
	}
	if len(options) != 1 || options[0].RemainingCapacity != 1 {
		t.Fatalf("booking options=%+v, want one option with one remaining capacity", options)
	}
	commands := []patientmanager.CreateBookingCommand{
		{ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"), Session: appointmentmanager.SessionMorning, OperationID: "41000000-0000-0000-0000-000000000011"},
		{ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"), Session: appointmentmanager.SessionMorning, OperationID: "41000000-0000-0000-0000-000000000012"},
	}

	type result struct {
		booking appointmentmanager.Booking
		command patientmanager.CreateBookingCommand
		err     error
	}
	results := make(chan result, len(commands))
	start := make(chan struct{})
	var wait sync.WaitGroup
	for _, command := range commands {
		command := command
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			booking, err := manager.CreateBooking(ctx, patient, command)
			results <- result{booking: booking, command: command, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	successes, full := 0, 0
	var winner result
	for value := range results {
		switch {
		case value.err == nil:
			successes++
			winner = value
		case errors.Is(value.err, appointmentmanager.ErrCapacityFull):
			full++
		default:
			t.Fatalf("unexpected concurrent booking error: %v", value.err)
		}
	}
	if successes != 1 || full != 1 {
		t.Fatalf("successes=%d full=%d, want 1 and 1", successes, full)
	}
	winner.command.RequestID = "retry-with-a-different-request-id"
	retried, err := manager.CreateBooking(ctx, patient, winner.command)
	if err != nil {
		t.Fatalf("retry idempotent booking: %v", err)
	}
	if retried.BookingID != winner.booking.BookingID {
		t.Fatalf("retry booking id=%s, want %s", retried.BookingID, winner.booking.BookingID)
	}

	var occupied, bookings int
	if err := store.db.QueryRowContext(ctx, `SELECT occupied_capacity FROM appointment_room_date_capacity WHERE room_id = ? AND service_date = ? AND session = 'morning'`, bookingTestRoom, serviceDate.Format("2006-01-02")).Scan(&occupied); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM appointment_bookings WHERE room_id = ? AND service_date = ?`, bookingTestRoom, serviceDate.Format("2006-01-02")).Scan(&bookings); err != nil {
		t.Fatal(err)
	}
	if occupied != 1 || bookings != 1 {
		t.Fatalf("occupied=%d bookings=%d, want 1 and 1", occupied, bookings)
	}
}

func seedBookingIntegrationData(t *testing.T, store *Store, ctx context.Context, weekday int) {
	t.Helper()
	now := time.Now().UTC()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO appointment_examination_items (id, owner_department_id, name, description, status, version, created_at, updated_at) VALUES (?, ?, 'Booking integration item', 'test', 'active', 1, ?, ?)`, []any{bookingTestItem, bookingTestDepartment, now, now}},
		{`INSERT INTO appointment_rooms (id, department_id, campus_id, building, floor_number, room_number, version, created_at, updated_at) VALUES (?, ?, ?, 'T', 1, '101', 1, ?, ?)`, []any{bookingTestRoom, bookingTestDepartment, bookingTestCampus, now, now}},
		{`INSERT INTO appointment_room_examination_items (id, room_id, item_id, status, version, created_at, updated_at) VALUES (?, ?, ?, 'active', 1, ?, ?)`, []any{bookingTestRelation, bookingTestRoom, bookingTestItem, now, now}},
		{`INSERT INTO appointment_room_weekly_windows (id, room_id, weekday, session, open_time, close_time, active_capacity, status, version, created_at, updated_at) VALUES (?, ?, ?, 'morning', '00:00:00', '23:59:59', 1, 'active', 1, ?, ?)`, []any{bookingTestRoomWindow, bookingTestRoom, weekday, now, now}},
		{`INSERT INTO appointment_item_weekly_windows (id, item_id, weekday, session, start_time, booking_cutoff_time, end_time, status, version, created_at, updated_at) VALUES (?, ?, ?, 'morning', '00:00:00', '23:59:58', '23:59:59', 'active', 1, ?, ?)`, []any{bookingTestItemWindow, bookingTestItem, weekday, now, now}},
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
}

func cleanupBookingIntegrationData(t *testing.T, store *Store, ctx context.Context) {
	t.Helper()
	statements := []string{
		`DELETE FROM appointment_booking_operations WHERE operator_account_id = '` + bookingTestPatient + `'`,
		`DELETE FROM appointment_bookings WHERE item_id = '` + bookingTestItem + `'`,
		`DELETE FROM appointment_room_date_capacity WHERE room_id = '` + bookingTestRoom + `'`,
		`DELETE FROM appointment_item_weekly_windows WHERE item_id = '` + bookingTestItem + `'`,
		`DELETE FROM appointment_room_weekly_windows WHERE room_id = '` + bookingTestRoom + `'`,
		`DELETE FROM appointment_room_examination_items WHERE item_id = '` + bookingTestItem + `'`,
		`DELETE FROM appointment_rooms WHERE id = '` + bookingTestRoom + `'`,
		`DELETE FROM appointment_examination_items WHERE id = '` + bookingTestItem + `'`,
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
}
