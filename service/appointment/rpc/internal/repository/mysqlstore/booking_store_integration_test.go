package mysqlstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"hospital/common/authn"
	contractauthz "hospital/contracts/authz"
	appointmentmanager "hospital/service/appointment/rpc/internal/manager/common"
	patientmanager "hospital/service/appointment/rpc/internal/manager/patient"
	staffmanager "hospital/service/appointment/rpc/internal/manager/staff"
	staffinput "hospital/service/appointment/rpc/internal/manager/staff/input"
)

const (
	bookingTestDepartment    = "41000000-0000-0000-0000-000000000001"
	bookingTestCampus        = "41000000-0000-0000-0000-000000000002"
	bookingTestItem          = "41000000-0000-0000-0000-000000000003"
	bookingTestRoom          = "41000000-0000-0000-0000-000000000004"
	bookingTestRelation      = "41000000-0000-0000-0000-000000000005"
	bookingTestRoomWindow    = "41000000-0000-0000-0000-000000000006"
	bookingTestItemWindow    = "41000000-0000-0000-0000-000000000007"
	bookingTestPatient       = "41000000-0000-0000-0000-000000000008"
	bookingTestRoomTwo       = "41000000-0000-0000-0000-000000000009"
	bookingTestRelationTwo   = "41000000-0000-0000-0000-000000000010"
	bookingTestRoomWindowTwo = "41000000-0000-0000-0000-000000000013"
	bookingTestPatientTwo    = "41000000-0000-0000-0000-000000000014"
	bookingTestStaff         = "41000000-0000-0000-0000-000000000015"
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
	if now.Minute() >= 58 && (now.Hour() == 11 || now.Hour() == 23) {
		t.Skip("not enough time remains in the service date for the booking cutoff")
	}
	serviceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekday := (int(serviceDate.Weekday())+6)%7 + 1
	session, openTime, cutoffTime, closeTime := bookingIntegrationWindow(now)

	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	cleanupBookingIntegrationData(t, store, ctx)
	defer cleanupBookingIntegrationData(t, store, ctx)
	seedBookingIntegrationData(t, store, ctx, weekday, session, openTime, cutoffTime, closeTime)

	manager, err := patientmanager.NewManager(store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	patients := []authn.Principal{
		{AccountID: bookingTestPatient, AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive},
		{AccountID: bookingTestPatientTwo, AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive},
	}
	options, _, _, err := manager.ListBookingOptions(ctx, patients[0], bookingTestItem)
	if err != nil {
		t.Fatalf("list booking options: %v", err)
	}
	if len(options) != 2 || options[0].RemainingCapacity != 1 || options[1].RemainingCapacity != 1 {
		t.Fatalf("booking options=%+v, want two options with one remaining capacity", options)
	}
	if options[0].EstimatedDurationMinutes != 35 || options[1].EstimatedDurationMinutes != 35 {
		t.Fatalf("booking option durations=%d/%d, want 35/35", options[0].EstimatedDurationMinutes, options[1].EstimatedDurationMinutes)
	}
	commands := []patientmanager.CreateBookingCommand{
		{ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"), Session: session, PatientDisplayName: "测试患者甲", PatientPhoneMasked: "134****0001", OperationID: "41000000-0000-0000-0000-000000000011"},
		{ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"), Session: session, PatientDisplayName: "测试患者乙", PatientPhoneMasked: "134****0002", OperationID: "41000000-0000-0000-0000-000000000012"},
	}

	type result struct {
		booking appointmentmanager.Booking
		command patientmanager.CreateBookingCommand
		patient authn.Principal
		err     error
	}
	results := make(chan result, len(commands))
	start := make(chan struct{})
	var wait sync.WaitGroup
	for index, command := range commands {
		command := command
		patient := patients[index]
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			booking, err := manager.CreateBooking(ctx, patient, command)
			results <- result{booking: booking, command: command, patient: patient, err: err}
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
	if winner.booking.Building != "T" || winner.booking.FloorNumber != 1 || winner.booking.RoomNumber != "101" {
		t.Fatalf("booking address=%q/%d/%q, want T/1/101", winner.booking.Building, winner.booking.FloorNumber, winner.booking.RoomNumber)
	}
	if winner.booking.EstimatedDurationMinutes != 35 {
		t.Fatalf("booking duration=%d, want snapshot 35", winner.booking.EstimatedDurationMinutes)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE appointment_examination_items SET estimated_duration_minutes = 45 WHERE id = ?`, bookingTestItem); err != nil {
		t.Fatal(err)
	}
	persisted, err := store.GetBooking(ctx, winner.booking.BookingID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.EstimatedDurationMinutes != 35 {
		t.Fatalf("persisted booking duration=%d after project update, want snapshot 35", persisted.EstimatedDurationMinutes)
	}
	winner.command.RequestID = "retry-with-a-different-request-id"
	retried, err := manager.CreateBooking(ctx, winner.patient, winner.command)
	if err != nil {
		t.Fatalf("retry idempotent booking: %v", err)
	}
	if retried.BookingID != winner.booking.BookingID {
		t.Fatalf("retry booking id=%s, want %s", retried.BookingID, winner.booking.BookingID)
	}

	var occupied, bookings int
	if err := store.db.QueryRowContext(ctx, `SELECT occupied_capacity FROM appointment_room_date_capacity WHERE room_id = ? AND service_date = ? AND session = ?`, bookingTestRoom, serviceDate.Format("2006-01-02"), session).Scan(&occupied); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM appointment_bookings WHERE room_id = ? AND service_date = ?`, bookingTestRoom, serviceDate.Format("2006-01-02")).Scan(&bookings); err != nil {
		t.Fatal(err)
	}
	if occupied != 1 || bookings != 1 {
		t.Fatalf("occupied=%d bookings=%d, want 1 and 1", occupied, bookings)
	}
}

func TestPatientSessionClaimBlocksMultipleRoomsUntilExaminationStarts(t *testing.T) {
	dataSource := os.Getenv("APPOINTMENT_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("APPOINTMENT_TEST_MYSQL_DSN is not set")
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(location)
	if now.Minute() >= 58 && (now.Hour() == 11 || now.Hour() == 23) {
		t.Skip("not enough time remains in the service date for the booking cutoff")
	}
	serviceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekday := (int(serviceDate.Weekday())+6)%7 + 1
	session, openTime, cutoffTime, closeTime := bookingIntegrationWindow(now)

	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	cleanupBookingIntegrationData(t, store, ctx)
	defer cleanupBookingIntegrationData(t, store, ctx)
	seedBookingIntegrationData(t, store, ctx, weekday, session, openTime, cutoffTime, closeTime)

	patientManager, err := patientmanager.NewManager(store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	staffManager, err := staffmanager.NewManager(store, store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	patient := authn.Principal{AccountID: bookingTestPatient, AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive}
	first, err := patientManager.CreateBooking(ctx, patient, patientmanager.CreateBookingCommand{
		ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"),
		Session: session, PatientDisplayName: "测试患者", PatientPhoneMasked: "134****0001", OperationID: "41000000-0000-0000-0000-000000000021",
	})
	if err != nil {
		t.Fatalf("create first booking: %v", err)
	}
	secondCommand := patientmanager.CreateBookingCommand{
		ItemID: bookingTestItem, RoomID: bookingTestRoomTwo, ServiceDate: serviceDate.Format("2006-01-02"),
		Session: session, PatientDisplayName: "测试患者", PatientPhoneMasked: "134****0001", OperationID: "41000000-0000-0000-0000-000000000022",
	}
	if _, err := patientManager.CreateBooking(ctx, patient, secondCommand); !errors.Is(err, appointmentmanager.ErrPatientSessionOccupied) {
		t.Fatalf("second room booking error=%v, want patient session occupied", err)
	}

	staff := authn.Principal{
		AccountID: bookingTestStaff, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
		Roles: []string{authn.RoleDepartmentDoctor}, DepartmentID: bookingTestDepartment,
		Permissions: []string{contractauthz.PermissionAppointmentUpdate},
	}
	started, err := staffManager.StartExamination(ctx, staff, staffinput.StartExamination{
		BookingID: first.BookingID, ExpectedVersion: first.Version,
		ActorDisplayName: "测试医生",
		Operation:        staffinput.Operation{OperationID: "41000000-0000-0000-0000-000000000023"},
	})
	if err != nil {
		t.Fatalf("start first examination: %v", err)
	}
	if started.Status != appointmentmanager.BookingStatusInProgress {
		t.Fatalf("started status=%s", started.Status)
	}
	second, err := patientManager.CreateBooking(ctx, patient, secondCommand)
	if err != nil {
		t.Fatalf("create after examination starts: %v", err)
	}

	var claims, bookings, occupied int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM appointment_patient_session_claims WHERE patient_account_id = ?`, bookingTestPatient).Scan(&claims); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM appointment_bookings WHERE patient_account_id = ?`, bookingTestPatient).Scan(&bookings); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(occupied_capacity), 0) FROM appointment_room_date_capacity WHERE room_id IN (?, ?)`, bookingTestRoom, bookingTestRoomTwo).Scan(&occupied); err != nil {
		t.Fatal(err)
	}
	if claims != 1 || bookings != 2 || occupied != 2 {
		t.Fatalf("claims=%d bookings=%d occupied=%d, want 1, 2, 2", claims, bookings, occupied)
	}
	var claimedBookingID string
	if err := store.db.QueryRowContext(ctx, `SELECT booking_id FROM appointment_patient_session_claims WHERE patient_account_id = ?`, bookingTestPatient).Scan(&claimedBookingID); err != nil {
		t.Fatal(err)
	}
	if claimedBookingID != second.BookingID {
		t.Fatalf("claimed booking=%s, want second booking %s", claimedBookingID, second.BookingID)
	}
}

func TestWeeklyQuotaIsNotReturnedAfterBookingDeletion(t *testing.T) {
	dataSource := os.Getenv("APPOINTMENT_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("APPOINTMENT_TEST_MYSQL_DSN is not set")
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(location)
	if now.Minute() >= 58 && (now.Hour() == 11 || now.Hour() == 23) {
		t.Skip("not enough time remains before the booking cutoff")
	}
	serviceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekday := (int(serviceDate.Weekday())+6)%7 + 1
	session, openTime, cutoffTime, closeTime := bookingIntegrationWindow(now)

	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	cleanupBookingIntegrationData(t, store, ctx)
	defer cleanupBookingIntegrationData(t, store, ctx)
	seedBookingIntegrationData(t, store, ctx, weekday, session, openTime, cutoffTime, closeTime)

	patientManager, err := patientmanager.NewManager(store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	staffManager, err := staffmanager.NewManager(store, store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	patient := authn.Principal{AccountID: bookingTestPatient, AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive}
	staff := authn.Principal{
		AccountID: bookingTestStaff, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
		Roles: []string{authn.RoleDepartmentDoctor}, DepartmentID: bookingTestDepartment,
		Permissions: []string{contractauthz.PermissionAppointmentCancel},
	}

	for index := 1; index <= 10; index++ {
		booking, createErr := patientManager.CreateBooking(ctx, patient, patientmanager.CreateBookingCommand{
			ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"),
			Session: session, PatientDisplayName: "测试患者", PatientPhoneMasked: "134****0001",
			OperationID: fmt.Sprintf("41000000-0000-0000-0001-%012d", index),
		})
		if createErr != nil {
			t.Fatalf("create booking %d: %v", index, createErr)
		}
		if _, deleteErr := staffManager.DeleteBooking(ctx, staff, staffinput.DeleteBooking{
			BookingID: booking.BookingID, OperationID: fmt.Sprintf("41000000-0000-0000-0002-%012d", index),
		}); deleteErr != nil {
			t.Fatalf("delete booking %d: %v", index, deleteErr)
		}
	}

	_, err = patientManager.CreateBooking(ctx, patient, patientmanager.CreateBookingCommand{
		ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"),
		Session: session, PatientDisplayName: "测试患者", PatientPhoneMasked: "134****0001",
		OperationID: "41000000-0000-0000-0003-000000000011",
	})
	if !errors.Is(err, appointmentmanager.ErrPatientWeeklyQuotaFull) {
		t.Fatalf("eleventh booking error=%v, want weekly quota exhausted", err)
	}
	var usedCount int64
	if err := store.db.QueryRowContext(ctx, `SELECT used_count FROM appointment_patient_weekly_quota_usage WHERE patient_account_id = ?`, bookingTestPatient).Scan(&usedCount); err != nil {
		t.Fatal(err)
	}
	if usedCount != 10 {
		t.Fatalf("used_count=%d, want 10", usedCount)
	}
}

func TestExpiredConfirmedBookingBecomesNoShow(t *testing.T) {
	dataSource := os.Getenv("APPOINTMENT_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("APPOINTMENT_TEST_MYSQL_DSN is not set")
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(location)
	if now.Hour() == 0 && now.Minute() == 0 && now.Second() <= 1 {
		t.Skip("test needs a timestamp after 00:00:01")
	}
	serviceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekday := (int(serviceDate.Weekday())+6)%7 + 1
	session, openTime, cutoffTime, closeTime := bookingIntegrationWindow(now)

	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	cleanupBookingIntegrationData(t, store, ctx)
	defer cleanupBookingIntegrationData(t, store, ctx)
	seedBookingIntegrationData(t, store, ctx, weekday, session, openTime, cutoffTime, closeTime)

	patientManager, err := patientmanager.NewManager(store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	patient := authn.Principal{AccountID: bookingTestPatient, AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive}
	booking, err := patientManager.CreateBooking(ctx, patient, patientmanager.CreateBookingCommand{
		ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"),
		Session: session, PatientDisplayName: "测试患者", PatientPhoneMasked: "134****0001",
		OperationID: "41000000-0000-0000-0004-000000000001",
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `
UPDATE appointment_bookings
SET room_open_time_snapshot = '00:00:00',
    item_start_time_snapshot = '00:00:00',
    item_cutoff_time_snapshot = '00:00:00',
    item_end_time_snapshot = '00:00:01'
WHERE id = ?`, booking.BookingID); err != nil {
		t.Fatal(err)
	}

	result, err := store.CleanupExpiredBookings(ctx, now, 10)
	if err != nil {
		t.Fatalf("mark no-show: %v", err)
	}
	if result.MarkedNoShow != 1 {
		t.Fatalf("marked_no_show=%d, want 1", result.MarkedNoShow)
	}
	updated, err := store.GetBooking(ctx, booking.BookingID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != appointmentmanager.BookingStatusNoShow {
		t.Fatalf("status=%s, want no_show", updated.Status)
	}
	var occupied, claims, quota int64
	if err := store.db.QueryRowContext(ctx, `SELECT occupied_capacity FROM appointment_room_date_capacity WHERE room_id = ? AND service_date = ? AND session = ?`, bookingTestRoom, serviceDate.Format("2006-01-02"), session).Scan(&occupied); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM appointment_patient_session_claims WHERE booking_id = ?`, booking.BookingID).Scan(&claims); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT used_count FROM appointment_patient_weekly_quota_usage WHERE patient_account_id = ?`, bookingTestPatient).Scan(&quota); err != nil {
		t.Fatal(err)
	}
	if occupied != 0 || claims != 0 || quota != 1 {
		t.Fatalf("occupied=%d claims=%d quota=%d, want 0, 0, 1", occupied, claims, quota)
	}
}

func TestExaminationReportPublishAndCorrectionPreserveHistory(t *testing.T) {
	dataSource := os.Getenv("APPOINTMENT_TEST_MYSQL_DSN")
	if dataSource == "" {
		t.Skip("APPOINTMENT_TEST_MYSQL_DSN is not set")
	}
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(location)
	if now.Minute() >= 58 && (now.Hour() == 11 || now.Hour() == 23) {
		t.Skip("not enough time remains in the service date")
	}
	serviceDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	weekday := (int(serviceDate.Weekday())+6)%7 + 1
	session, openTime, cutoffTime, closeTime := bookingIntegrationWindow(now)
	store, err := New(dataSource)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	cleanupBookingIntegrationData(t, store, ctx)
	defer cleanupBookingIntegrationData(t, store, ctx)
	seedBookingIntegrationData(t, store, ctx, weekday, session, openTime, cutoffTime, closeTime)

	patientManager, err := patientmanager.NewManager(store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	staffManager, err := staffmanager.NewManager(store, store, store, store, store, nil)
	if err != nil {
		t.Fatal(err)
	}
	patient := authn.Principal{AccountID: bookingTestPatient, AccountType: authn.AccountTypePatient, Status: authn.AccountStatusActive}
	staff := authn.Principal{
		AccountID: bookingTestStaff, AccountType: authn.AccountTypeStaff, Status: authn.AccountStatusActive,
		Roles: []string{authn.RoleDepartmentDoctor}, DepartmentID: bookingTestDepartment,
		Permissions: []string{contractauthz.PermissionAppointmentUpdate, contractauthz.PermissionReportRead, contractauthz.PermissionReportPublish, contractauthz.PermissionReportCorrect},
	}
	booking, err := patientManager.CreateBooking(ctx, patient, patientmanager.CreateBookingCommand{ItemID: bookingTestItem, RoomID: bookingTestRoom, ServiceDate: serviceDate.Format("2006-01-02"), Session: session, PatientDisplayName: "测试患者", PatientPhoneMasked: "134****0001", OperationID: "41000000-0000-0000-0000-000000000031"})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	booking, err = staffManager.StartExamination(ctx, staff, staffinput.StartExamination{BookingID: booking.BookingID, ExpectedVersion: booking.Version, ActorDisplayName: "测试医生", Operation: staffinput.Operation{OperationID: "41000000-0000-0000-0000-000000000033"}})
	if err != nil {
		t.Fatalf("start examination: %v", err)
	}
	draft, err := staffManager.SaveExaminationReportDraft(ctx, staff, staffinput.SaveReportDraft{BookingID: booking.BookingID, Content: appointmentmanager.ReportContent{ObjectiveFindings: "draft finding"}, ExpectedReportVersion: 0, ActorDisplayName: "测试医生", Operation: staffinput.Operation{OperationID: "41000000-0000-0000-0000-000000000034"}})
	if err != nil {
		t.Fatalf("save report draft: %v", err)
	}
	if _, err := patientManager.GetMyExaminationReport(ctx, patient, booking.BookingID); !errors.Is(err, appointmentmanager.ErrNotFound) {
		t.Fatalf("patient draft read error=%v, want not found", err)
	}
	report, err := staffManager.CompleteAndPublishExaminationReport(ctx, staff, staffinput.CompleteAndPublishReport{BookingID: booking.BookingID, Content: appointmentmanager.ReportContent{ObjectiveFindings: "published finding", Impression: "initial impression"}, ExpectedBookingVersion: booking.Version, ExpectedReportVersion: draft.Version, ActorDisplayName: "测试医生", DepartmentName: "测试科室", CampusName: "测试院区", Operation: staffinput.Operation{OperationID: "41000000-0000-0000-0000-000000000035"}})
	if err != nil {
		t.Fatalf("complete and publish: %v", err)
	}
	if report.CurrentVersion == nil || report.CurrentVersion.VersionNo != 1 || report.Status != appointmentmanager.ReportStatusPublished {
		t.Fatalf("published report=%+v", report)
	}
	patientReport, err := patientManager.GetMyExaminationReport(ctx, patient, booking.BookingID)
	if err != nil {
		t.Fatalf("patient get report: %v", err)
	}
	if patientReport.CurrentVersion.Impression != "initial impression" {
		t.Fatalf("initial impression=%q", patientReport.CurrentVersion.Impression)
	}
	corrected, err := staffManager.CorrectExaminationReport(ctx, staff, staffinput.CorrectReport{ReportID: report.ReportID, Content: appointmentmanager.ReportContent{ObjectiveFindings: "corrected finding", Impression: "corrected impression"}, CorrectionReason: "修正录入错误", ExpectedReportVersion: report.Version, ActorDisplayName: "测试医生", Operation: staffinput.Operation{OperationID: "41000000-0000-0000-0000-000000000036"}})
	if err != nil {
		t.Fatalf("correct report: %v", err)
	}
	if corrected.CurrentVersion == nil || corrected.CurrentVersion.VersionNo != 2 || corrected.CurrentVersion.CorrectionReason == "" {
		t.Fatalf("corrected report=%+v", corrected)
	}
	versions, err := staffManager.ListExaminationReportVersions(ctx, staff, report.ReportID)
	if err != nil {
		t.Fatalf("list versions: %v", err)
	}
	if len(versions) != 2 || versions[0].Status != appointmentmanager.ReportVersionStatusPublished || versions[1].Status != appointmentmanager.ReportVersionStatusSuperseded || versions[1].Impression != "initial impression" {
		t.Fatalf("versions=%+v", versions)
	}
	var occupied int
	if err := store.db.QueryRowContext(ctx, `SELECT occupied_capacity FROM appointment_room_date_capacity WHERE room_id = ? AND service_date = ? AND session = ?`, bookingTestRoom, serviceDate.Format("2006-01-02"), session).Scan(&occupied); err != nil {
		t.Fatal(err)
	}
	if occupied != 0 {
		t.Fatalf("occupied capacity=%d, want 0 after completion", occupied)
	}
}

func bookingIntegrationWindow(now time.Time) (appointmentmanager.Session, string, string, string) {
	if now.Hour() < 12 {
		return appointmentmanager.SessionMorning, "00:00:00", "11:59:58", "11:59:59"
	}
	return appointmentmanager.SessionAfternoon, "12:00:00", "23:59:58", "23:59:59"
}

func seedBookingIntegrationData(t *testing.T, store *Store, ctx context.Context, weekday int, session appointmentmanager.Session, openTime, cutoffTime, closeTime string) {
	t.Helper()
	now := time.Now().UTC()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO appointment_examination_items
			(id, owner_department_id, name, description, estimated_duration_minutes,
             report_template_objective_findings, report_template_impression,
             report_template_recommendation, report_template_notes, report_template_version,
             status, version, created_at, updated_at)
			VALUES (?, ?, 'Booking integration item', 'test', 35, '', '', '', '', 0, 'active', 1, ?, ?)`, []any{bookingTestItem, bookingTestDepartment, now, now}},
		{`INSERT INTO appointment_rooms (id, department_id, campus_id, building, floor_number, room_number, version, created_at, updated_at) VALUES (?, ?, ?, 'T', 1, '101', 1, ?, ?)`, []any{bookingTestRoom, bookingTestDepartment, bookingTestCampus, now, now}},
		{`INSERT INTO appointment_rooms (id, department_id, campus_id, building, floor_number, room_number, version, created_at, updated_at) VALUES (?, ?, ?, 'T', 1, '102', 1, ?, ?)`, []any{bookingTestRoomTwo, bookingTestDepartment, bookingTestCampus, now, now}},
		{`INSERT INTO appointment_room_examination_items (id, room_id, item_id, status, version, created_at, updated_at) VALUES (?, ?, ?, 'active', 1, ?, ?)`, []any{bookingTestRelation, bookingTestRoom, bookingTestItem, now, now}},
		{`INSERT INTO appointment_room_examination_items (id, room_id, item_id, status, version, created_at, updated_at) VALUES (?, ?, ?, 'active', 1, ?, ?)`, []any{bookingTestRelationTwo, bookingTestRoomTwo, bookingTestItem, now, now}},
		{`INSERT INTO appointment_room_weekly_windows (id, room_id, weekday, session, open_time, close_time, active_capacity, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 1, 'active', 1, ?, ?)`, []any{bookingTestRoomWindow, bookingTestRoom, weekday, session, openTime, closeTime, now, now}},
		{`INSERT INTO appointment_room_weekly_windows (id, room_id, weekday, session, open_time, close_time, active_capacity, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 1, 'active', 1, ?, ?)`, []any{bookingTestRoomWindowTwo, bookingTestRoomTwo, weekday, session, openTime, closeTime, now, now}},
		{`INSERT INTO appointment_item_weekly_windows (id, item_id, weekday, session, start_time, booking_cutoff_time, end_time, status, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'active', 1, ?, ?)`, []any{bookingTestItemWindow, bookingTestItem, weekday, session, openTime, cutoffTime, closeTime, now, now}},
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
		`UPDATE appointment_examination_reports SET current_version_id = NULL WHERE booking_id IN (SELECT id FROM appointment_bookings WHERE item_id = '` + bookingTestItem + `')`,
		`DELETE FROM appointment_examination_report_versions WHERE report_id IN (SELECT id FROM appointment_examination_reports WHERE item_id = '` + bookingTestItem + `')`,
		`DELETE FROM appointment_examination_reports WHERE item_id = '` + bookingTestItem + `'`,
		`DELETE FROM appointment_booking_operations WHERE operator_account_id = '` + bookingTestPatient + `'`,
		`DELETE FROM appointment_booking_operations WHERE operator_account_id IN ('` + bookingTestPatientTwo + `', '` + bookingTestStaff + `')`,
		`DELETE FROM appointment_patient_session_claims WHERE patient_account_id IN ('` + bookingTestPatient + `', '` + bookingTestPatientTwo + `')`,
		`DELETE FROM appointment_patient_weekly_quota_usage WHERE patient_account_id IN ('` + bookingTestPatient + `', '` + bookingTestPatientTwo + `')`,
		`DELETE FROM appointment_bookings WHERE item_id = '` + bookingTestItem + `'`,
		`DELETE FROM appointment_room_date_capacity WHERE room_id IN ('` + bookingTestRoom + `', '` + bookingTestRoomTwo + `')`,
		`DELETE FROM appointment_item_weekly_windows WHERE item_id = '` + bookingTestItem + `'`,
		`DELETE FROM appointment_room_weekly_windows WHERE room_id IN ('` + bookingTestRoom + `', '` + bookingTestRoomTwo + `')`,
		`DELETE FROM appointment_room_examination_items WHERE item_id = '` + bookingTestItem + `'`,
		`DELETE FROM appointment_rooms WHERE id IN ('` + bookingTestRoom + `', '` + bookingTestRoomTwo + `')`,
		`DELETE FROM appointment_examination_items WHERE id = '` + bookingTestItem + `'`,
	}
	for _, statement := range statements {
		if _, err := store.db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
}
