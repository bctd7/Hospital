ALTER TABLE appointment_examination_reports
    DROP FOREIGN KEY fk_appointment_examination_reports_current_version;
DROP TABLE IF EXISTS appointment_examination_report_versions;
DROP TABLE IF EXISTS appointment_examination_reports;
DROP TABLE IF EXISTS appointment_booking_operations;
DROP TABLE IF EXISTS appointment_patient_session_claims;
DROP TABLE IF EXISTS appointment_bookings;
DROP TABLE IF EXISTS appointment_patient_weekly_quota_usage;
DROP TABLE IF EXISTS appointment_room_date_capacity;
DROP TABLE IF EXISTS appointment_resource_audit;
DROP TABLE IF EXISTS appointment_resource_operations;
DROP TABLE IF EXISTS appointment_item_weekly_windows;
DROP TABLE IF EXISTS appointment_room_weekly_windows;
DROP TABLE IF EXISTS appointment_room_examination_items;
DROP TABLE IF EXISTS appointment_rooms;
DROP TABLE IF EXISTS appointment_examination_item_audit;
DROP TABLE IF EXISTS appointment_examination_item_operations;
DROP TABLE IF EXISTS appointment_examination_items;
