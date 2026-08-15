SET time_zone = '+08:00';

USE hospital_identity;

SET @hospital = (SELECT id FROM identity_organization_units WHERE unit_type = 'hospital' AND status = 'active' ORDER BY id LIMIT 1);
SET @campus_main = '11000000-0000-4000-8000-000000000001';
SET @campus_east = '11000000-0000-4000-8000-000000000002';
SET @dept_radiology_main = '12000000-0000-4000-8000-000000000001';
SET @dept_ultrasound_main = '12000000-0000-4000-8000-000000000002';
SET @dept_radiology_east = '12000000-0000-4000-8000-000000000003';
SET @dept_laboratory_east = '12000000-0000-4000-8000-000000000004';

SET @account_doctor_1 = '20000000-0000-4000-8000-000000000002';
SET @account_doctor_2 = '20000000-0000-4000-8000-000000000003';
-- 部署配置中 153 开头的真实超级管理员同时作为报告测试患者。
-- 不能在这里另造同手机号患者，否则登录账号与报告所属账号会是两个不同 UUID。
SET @account_patient_1_fallback = '21000000-0000-4000-8000-000000000001';
SET @account_patient_1 = COALESCE(
    (SELECT account_id FROM identity_account_phones WHERE phone_fingerprint = @phone_patient_1 LIMIT 1),
    @account_patient_1_fallback
);
SET @account_patient_2 = '21000000-0000-4000-8000-000000000002';
SET @account_patient_3 = '21000000-0000-4000-8000-000000000003';
SET @account_patient_4 = '21000000-0000-4000-8000-000000000004';

INSERT INTO identity_organization_units (id, parent_id, unit_type, code, name, status, version) VALUES
(@campus_main, @hospital, 'campus', 'MAIN', '总院区', 'active', 1),
(@campus_east, @hospital, 'campus', 'EAST', '东院区', 'active', 1),
(@dept_radiology_main, @campus_main, 'department', 'RAD-MAIN', '放射科', 'active', 1),
(@dept_ultrasound_main, @campus_main, 'department', 'US-MAIN', '超声科', 'active', 1),
(@dept_radiology_east, @campus_east, 'department', 'RAD-EAST', '放射科', 'active', 1),
(@dept_laboratory_east, @campus_east, 'department', 'LAB-EAST', '检验科', 'active', 1);

INSERT INTO identity_accounts (id, account_type, status, authorization_version, management_version) VALUES
(@account_doctor_1, 'staff', 'active', 1, 1),
(@account_doctor_2, 'staff', 'active', 1, 1),
(@account_patient_2, 'patient', 'active', 1, 1),
(@account_patient_3, 'patient', 'active', 1, 1),
(@account_patient_4, 'patient', 'active', 1, 1);

INSERT INTO identity_accounts (id, account_type, status, authorization_version, management_version)
SELECT @account_patient_1, 'patient', 'active', 1, 1
WHERE NOT EXISTS (SELECT 1 FROM identity_accounts WHERE id = @account_patient_1);

INSERT INTO identity_account_profiles (account_id, nickname) VALUES
(@account_doctor_1, '陈医生'),
(@account_doctor_2, '周医生'),
(@account_patient_2, '李敏'),
(@account_patient_3, '王芳'),
(@account_patient_4, '赵强');

INSERT IGNORE INTO identity_account_profiles (account_id, nickname)
VALUES (@account_patient_1, '体验管理员');

INSERT IGNORE INTO identity_account_profiles (account_id, nickname)
SELECT ar.account_id, '体验管理员'
FROM identity_account_roles ar
JOIN identity_roles r ON r.id = ar.role_id
WHERE r.code = 'super_admin';

INSERT INTO identity_account_phones
    (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at)
VALUES
(@account_doctor_1, @phone_doctor_1, '138****0001', 'verified', 'admin', NOW(3)),
(@account_doctor_2, @phone_doctor_2, '138****0002', 'verified', 'admin', NOW(3)),
(@account_patient_2, @phone_patient_2, '139****0002', 'verified', 'sms', NOW(3)),
(@account_patient_3, @phone_patient_3, '139****0003', 'verified', 'sms', NOW(3)),
(@account_patient_4, @phone_patient_4, '139****0004', 'verified', 'sms', NOW(3));

INSERT INTO identity_account_phones
    (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at)
SELECT @account_patient_1, @phone_patient_1, @phone_patient_1_masked, 'verified', 'sms', NOW(3)
WHERE NOT EXISTS (
    SELECT 1 FROM identity_account_phones WHERE phone_fingerprint = @phone_patient_1
);

INSERT INTO identity_staff_profiles
    (account_id, department_id, staff_no, display_name, description, staff_status)
VALUES
(@account_doctor_1, @dept_radiology_main, 'D1001', '陈医生', '放射科主治医生', 'active'),
(@account_doctor_2, @dept_ultrasound_main, 'D2001', '周医生', '超声科主治医生', 'active');

INSERT INTO identity_account_roles (account_id, role_id)
SELECT @account_doctor_1, id FROM identity_roles WHERE code = 'department_doctor';
INSERT INTO identity_account_roles (account_id, role_id)
SELECT @account_doctor_2, id FROM identity_roles WHERE code = 'department_doctor';

USE hospital_appointment;

SET @today = CURDATE();
SET @tomorrow = DATE_ADD(@today, INTERVAL 1 DAY);
SET @yesterday = DATE_SUB(@today, INTERVAL 1 DAY);
SET @two_days_ago = DATE_SUB(@today, INTERVAL 2 DAY);
SET @three_days_ago = DATE_SUB(@today, INTERVAL 3 DAY);
-- 业务日期和窗口按医院所在的东八区计算；所有“事件时刻”按服务约定存 UTC。
-- Appointment 的 MySQL DSN 使用 loc=UTC，混写本地 DATETIME 会让提醒判断整体偏移 8 小时。
SET @local_now = NOW(3);
SET @now = UTC_TIMESTAMP(3);
SET @current_session = CASE WHEN CURTIME() < '12:00:00' THEN 'morning' ELSE 'afternoon' END;
SET @current_open_time = CASE WHEN @current_session = 'morning' THEN '00:00:00' ELSE '12:00:00' END;
SET @current_start_time = CASE
    WHEN @current_session = 'morning' THEN GREATEST(CAST('00:00:00' AS TIME), TIME(DATE_SUB(@local_now, INTERVAL 30 MINUTE)))
    ELSE GREATEST(CAST('12:00:00' AS TIME), TIME(DATE_SUB(@local_now, INTERVAL 30 MINUTE)))
END;
SET @current_cutoff_time = CASE
    WHEN @current_session = 'morning' THEN LEAST(CAST('11:59:40' AS TIME), TIME(DATE_ADD(@local_now, INTERVAL 15 MINUTE)))
    ELSE LEAST(CAST('23:59:40' AS TIME), TIME(DATE_ADD(@local_now, INTERVAL 15 MINUTE)))
END;
SET @current_end_time = CASE
    WHEN @current_session = 'morning' THEN LEAST(CAST('11:59:59' AS TIME), TIME(DATE_ADD(@local_now, INTERVAL 30 MINUTE)))
    ELSE LEAST(CAST('23:59:59' AS TIME), TIME(DATE_ADD(@local_now, INTERVAL 30 MINUTE)))
END;
SET @current_close_time = CASE WHEN @current_session = 'morning' THEN '11:59:59' ELSE '23:59:59' END;
SET @current_session_started_at = CASE
    WHEN @current_session = 'morning' THEN TIMESTAMP(@today, '00:00:00')
    ELSE TIMESTAMP(@today, '12:00:00')
END;
SET @current_session_started_at_utc = CONVERT_TZ(@current_session_started_at, '+08:00', '+00:00');

SET @item_ct = '30000000-0000-4000-8000-000000000001';
SET @item_xray = '30000000-0000-4000-8000-000000000002';
SET @item_urgent_ct = '30000000-0000-4000-8000-000000000003';
SET @item_mri_disabled = '30000000-0000-4000-8000-000000000004';
SET @item_ultrasound = '30000000-0000-4000-8000-000000000005';
SET @item_east_ct = '30000000-0000-4000-8000-000000000006';
SET @item_blood = '30000000-0000-4000-8000-000000000007';

INSERT INTO appointment_examination_items
    (id, owner_department_id, name, description, estimated_duration_minutes,
     report_template_objective_findings, report_template_impression,
     report_template_recommendation, report_template_notes,
     report_template_version, status, version, created_at, updated_at)
VALUES
(@item_ct, @dept_radiology_main, '胸部CT平扫', '胸部低剂量CT平扫，用于肺部常规筛查。', 20, '双肺纹理及密度：\n纵隔与胸膜：', '胸部CT检查结论：', '请结合临床，必要时复查。', '', 1, 'active', 1, @now, @now),
(@item_xray, @dept_radiology_main, '胸部X线正侧位', '胸部数字化摄影正侧位检查。', 15, '胸廓对称性：\n心肺影像：', '胸部X线检查结论：', '', '', 1, 'active', 1, @now, @now),
(@item_urgent_ct, @dept_radiology_main, '当日急诊CT', '用于测试当天时间段内开始检查与报告流程。', 30, '扫描部位及主要所见：', '急诊CT检查结论：', '建议结合急诊临床表现。', '综合测试项目', 1, 'active', 1, @now, @now),
(@item_mri_disabled, @dept_radiology_main, '头颅MRI（暂未开放）', '用于测试停用项目展示。', 45, '', '', '', '', 0, 'disabled', 1, @now, @now),
(@item_ultrasound, @dept_ultrasound_main, '腹部彩超', '肝胆胰脾肾常规超声检查。', 25, '肝胆胰脾肾超声所见：', '腹部超声检查结论：', '', '', 1, 'active', 1, @now, @now),
(@item_east_ct, @dept_radiology_east, '胸部CT平扫', '东院区胸部CT平扫。', 20, '双肺及纵隔所见：', '胸部CT检查结论：', '', '', 1, 'active', 1, @now, @now),
(@item_blood, @dept_laboratory_east, '血常规', '静脉血常规检查。', 10, '白细胞：\n红细胞：\n血小板：', '血常规检查结论：', '', '', 1, 'active', 1, @now, @now);

SET @room_ct201 = '31000000-0000-4000-8000-000000000001';
SET @room_ct202 = '31000000-0000-4000-8000-000000000002';
SET @room_dr101 = '31000000-0000-4000-8000-000000000003';
SET @room_us301 = '31000000-0000-4000-8000-000000000004';
SET @room_east_ct105 = '31000000-0000-4000-8000-000000000005';
SET @room_lab201 = '31000000-0000-4000-8000-000000000006';

INSERT INTO appointment_rooms
    (id, department_id, campus_id, building, floor_number, room_number, version, created_at, updated_at)
VALUES
(@room_ct201, @dept_radiology_main, @campus_main, '影像楼', 2, 'CT201', 1, @now, @now),
(@room_ct202, @dept_radiology_main, @campus_main, '影像楼', 2, 'CT202', 1, @now, @now),
(@room_dr101, @dept_radiology_main, @campus_main, '影像楼', 1, 'DR101', 1, @now, @now),
(@room_us301, @dept_ultrasound_main, @campus_main, '门诊楼', 3, 'US301', 1, @now, @now),
(@room_east_ct105, @dept_radiology_east, @campus_east, '医技楼', 1, 'CT105', 1, @now, @now),
(@room_lab201, @dept_laboratory_east, @campus_east, '医技楼', 2, 'LAB201', 1, @now, @now);

INSERT INTO appointment_room_examination_items
    (id, room_id, item_id, status, version, created_at, updated_at)
VALUES
('32000000-0000-4000-8000-000000000001', @room_ct201, @item_ct, 'active', 1, @now, @now),
('32000000-0000-4000-8000-000000000002', @room_ct202, @item_ct, 'active', 1, @now, @now),
('32000000-0000-4000-8000-000000000003', @room_dr101, @item_xray, 'active', 1, @now, @now),
('32000000-0000-4000-8000-000000000004', @room_ct201, @item_urgent_ct, 'active', 1, @now, @now),
('32000000-0000-4000-8000-000000000005', @room_us301, @item_ultrasound, 'active', 1, @now, @now),
('32000000-0000-4000-8000-000000000006', @room_east_ct105, @item_east_ct, 'active', 1, @now, @now),
('32000000-0000-4000-8000-000000000007', @room_lab201, @item_blood, 'active', 1, @now, @now),
('32000000-0000-4000-8000-000000000008', @room_ct202, @item_mri_disabled, 'disabled', 1, @now, @now);

INSERT INTO appointment_room_weekly_windows
    (id, room_id, weekday, session, open_time, close_time, active_capacity, status, version, created_at, updated_at)
SELECT UUID(), room.id, day.weekday, session.name,
       CASE WHEN room.id = @room_ct201 AND session.name = @current_session THEN @current_open_time
            WHEN session.name = 'morning' THEN '08:00:00' ELSE '12:00:00' END,
       CASE WHEN session.name = 'morning' THEN '12:00:00'
            WHEN room.id = @room_ct201 THEN '23:59:59' ELSE '18:00:00' END,
       CASE WHEN room.id IN (@room_ct201, @room_ct202) THEN 6 ELSE 4 END,
       'active', 1, @now, @now
FROM appointment_rooms room
CROSS JOIN (SELECT 1 weekday UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7) day
CROSS JOIN (SELECT 'morning' name UNION ALL SELECT 'afternoon') session;

INSERT INTO appointment_item_weekly_windows
    (id, item_id, weekday, session, start_time, booking_cutoff_time, end_time, status, version, created_at, updated_at)
SELECT UUID(), item.id, day.weekday, session.name,
       CASE WHEN item.id = @item_urgent_ct THEN @current_start_time WHEN session.name = 'morning' THEN '09:00:00' ELSE '14:00:00' END,
       CASE WHEN item.id = @item_urgent_ct THEN @current_cutoff_time WHEN session.name = 'morning' THEN '11:30:00' ELSE '16:30:00' END,
       CASE WHEN item.id = @item_urgent_ct THEN @current_end_time WHEN session.name = 'morning' THEN '12:00:00' ELSE '17:00:00' END,
       'active', 1, @now, @now
FROM appointment_examination_items item
CROSS JOIN (SELECT 1 weekday UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7) day
CROSS JOIN (SELECT 'morning' name UNION ALL SELECT 'afternoon') session
WHERE item.status = 'active' AND (item.id <> @item_urgent_ct OR session.name = @current_session);

SET @booking_confirmed_now = '40000000-0000-4000-8000-000000000001';
SET @booking_confirmed_future = '40000000-0000-4000-8000-000000000002';
SET @booking_no_show = '40000000-0000-4000-8000-000000000003';
SET @booking_in_progress = '40000000-0000-4000-8000-000000000004';
SET @booking_completed = '40000000-0000-4000-8000-000000000005';
SET @booking_corrected = '40000000-0000-4000-8000-000000000006';
SET @booking_report_overdue = '40000000-0000-4000-8000-000000000007';
SET @booking_canceled = '40000000-0000-4000-8000-000000000008';

INSERT INTO appointment_bookings
    (id, patient_account_id, patient_display_name_snapshot, patient_phone_masked_snapshot,
     patient_phone_last4_snapshot, department_id, item_id, room_id, service_date, session, status,
     room_open_time_snapshot, room_close_time_snapshot, item_start_time_snapshot,
     item_end_time_snapshot, item_cutoff_time_snapshot, estimated_duration_minutes_snapshot, started_at, started_by, started_by_display_name_snapshot,
     completed_at, completed_by, completed_by_display_name_snapshot,
     version, created_at, updated_at)
VALUES
(@booking_confirmed_now, @account_patient_1, '体验管理员', @phone_patient_1_masked, @phone_patient_1_last4, @dept_radiology_main, @item_urgent_ct, @room_ct201, @today, @current_session, 'confirmed', @current_open_time, @current_close_time, @current_start_time, @current_end_time, @current_cutoff_time, 30, NULL, NULL, NULL, NULL, NULL, NULL, 1, DATE_SUB(@now, INTERVAL 2 HOUR), @now),
(@booking_confirmed_future, @account_patient_2, '李敏', '139****0002', '0002', @dept_radiology_main, @item_ct, @room_ct202, @tomorrow, 'morning', 'confirmed', '08:00:00', '12:00:00', '09:00:00', '12:00:00', '11:30:00', 20, NULL, NULL, NULL, NULL, NULL, NULL, 1, DATE_SUB(@now, INTERVAL 1 HOUR), @now),
(@booking_no_show, @account_patient_3, '王芳', '139****0003', '0003', @dept_radiology_main, @item_ct, @room_ct201, @yesterday, 'morning', 'no_show', '08:00:00', '12:00:00', '09:00:00', '12:00:00', '11:30:00', 20, NULL, NULL, NULL, NULL, NULL, NULL, 2, DATE_SUB(@now, INTERVAL 1 DAY), @now),
(@booking_in_progress, @account_patient_4, '赵强', '139****0004', '0004', @dept_radiology_main, @item_urgent_ct, @room_ct201, @today, @current_session, 'in_progress', @current_open_time, @current_close_time, @current_start_time, @current_end_time, @current_cutoff_time, 30, IF(TIMESTAMPDIFF(SECOND, @current_session_started_at_utc, DATE_SUB(@now, INTERVAL 20 MINUTE)) > 0, DATE_SUB(@now, INTERVAL 20 MINUTE), CAST(@current_session_started_at_utc AS DATETIME)), @account_doctor_1, '陈医生', NULL, NULL, NULL, 2, DATE_SUB(@now, INTERVAL 4 HOUR), @now),
(@booking_completed, @account_patient_1, '体验管理员', @phone_patient_1_masked, @phone_patient_1_last4, @dept_ultrasound_main, @item_ultrasound, @room_us301, @yesterday, 'morning', 'completed', '08:00:00', '12:00:00', '09:00:00', '12:00:00', '11:30:00', 25, CONVERT_TZ(TIMESTAMP(@yesterday, '09:10:00'), '+08:00', '+00:00'), @account_doctor_2, '周医生', CONVERT_TZ(TIMESTAMP(@yesterday, '09:50:00'), '+08:00', '+00:00'), @account_doctor_2, '周医生', 3, CONVERT_TZ(TIMESTAMP(@two_days_ago, '15:00:00'), '+08:00', '+00:00'), @now),
(@booking_corrected, @account_patient_1, '体验管理员', @phone_patient_1_masked, @phone_patient_1_last4, @dept_radiology_main, @item_xray, @room_dr101, @two_days_ago, 'morning', 'completed', '08:00:00', '12:00:00', '09:00:00', '12:00:00', '11:30:00', 15, CONVERT_TZ(TIMESTAMP(@two_days_ago, '09:20:00'), '+08:00', '+00:00'), @account_doctor_1, '陈医生', CONVERT_TZ(TIMESTAMP(@two_days_ago, '10:10:00'), '+08:00', '+00:00'), @account_doctor_1, '陈医生', 3, CONVERT_TZ(TIMESTAMP(@three_days_ago, '15:00:00'), '+08:00', '+00:00'), @now),
(@booking_report_overdue, @account_patient_4, '赵强', '139****0004', '0004', @dept_radiology_main, @item_xray, @room_dr101, @yesterday, 'morning', 'in_progress', '08:00:00', '12:00:00', '09:00:00', '12:00:00', '11:30:00', 15, CONVERT_TZ(TIMESTAMP(@yesterday, '09:15:00'), '+08:00', '+00:00'), @account_doctor_1, '陈医生', NULL, NULL, NULL, 2, CONVERT_TZ(TIMESTAMP(@two_days_ago, '16:00:00'), '+08:00', '+00:00'), @now),
(@booking_canceled, @account_patient_2, '李敏', '139****0002', '0002', @dept_radiology_main, @item_ct, @room_ct202, @tomorrow, 'afternoon', 'canceled', '12:00:00', '18:00:00', '14:00:00', '17:00:00', '16:30:00', 20, NULL, NULL, NULL, NULL, NULL, NULL, 2, DATE_SUB(@now, INTERVAL 3 HOUR), DATE_SUB(@now, INTERVAL 2 HOUR));

INSERT INTO appointment_booking_operations
    (operation_id, operator_account_id, booking_id, action, request_fingerprint, result_data, created_at)
VALUES
('42000000-0000-4000-8000-000000000001', @account_patient_2, @booking_canceled,
 'delete_by_patient', REPEAT('a', 64), JSON_OBJECT('booking_id', @booking_canceled, 'deleted', TRUE), DATE_SUB(@now, INTERVAL 2 HOUR));

INSERT INTO appointment_patient_weekly_quota_usage
    (patient_account_id, week_start_date, used_count, version, created_at, updated_at)
SELECT patient_account_id,
       DATE_SUB(service_date, INTERVAL WEEKDAY(service_date) DAY),
       COUNT(*), 1, @now, @now
FROM appointment_bookings
GROUP BY patient_account_id, DATE_SUB(service_date, INTERVAL WEEKDAY(service_date) DAY);

INSERT INTO appointment_patient_session_claims
    (patient_account_id, service_date, session, booking_id)
VALUES
(@account_patient_1, @today, @current_session, @booking_confirmed_now),
(@account_patient_2, @tomorrow, 'morning', @booking_confirmed_future);

INSERT INTO appointment_room_date_capacity
    (id, room_id, service_date, session, total_capacity, occupied_capacity,
     room_window_id, room_window_version, version, created_at, updated_at)
SELECT '41000000-0000-4000-8000-000000000001', @room_ct201, @today, @current_session, 6, 2,
       rw.id, rw.version, 1, @now, @now
FROM appointment_room_weekly_windows rw
WHERE rw.room_id = @room_ct201 AND rw.weekday = WEEKDAY(@today) + 1 AND rw.session = @current_session;
INSERT INTO appointment_room_date_capacity
    (id, room_id, service_date, session, total_capacity, occupied_capacity,
     room_window_id, room_window_version, version, created_at, updated_at)
SELECT '41000000-0000-4000-8000-000000000002', @room_ct202, @tomorrow, 'morning', 6, 1,
       rw.id, rw.version, 1, @now, @now
FROM appointment_room_weekly_windows rw
WHERE rw.room_id = @room_ct202 AND rw.weekday = WEEKDAY(@tomorrow) + 1 AND rw.session = 'morning';
INSERT INTO appointment_room_date_capacity
    (id, room_id, service_date, session, total_capacity, occupied_capacity,
     room_window_id, room_window_version, version, created_at, updated_at)
SELECT '41000000-0000-4000-8000-000000000003', @room_us301, @yesterday, 'morning', 4, 0,
       rw.id, rw.version, 1, @now, @now
FROM appointment_room_weekly_windows rw
WHERE rw.room_id = @room_us301 AND rw.weekday = WEEKDAY(@yesterday) + 1 AND rw.session = 'morning';
INSERT INTO appointment_room_date_capacity
    (id, room_id, service_date, session, total_capacity, occupied_capacity,
     room_window_id, room_window_version, version, created_at, updated_at)
SELECT '41000000-0000-4000-8000-000000000004', @room_dr101, @two_days_ago, 'morning', 4, 0,
       rw.id, rw.version, 1, @now, @now
FROM appointment_room_weekly_windows rw
WHERE rw.room_id = @room_dr101 AND rw.weekday = WEEKDAY(@two_days_ago) + 1 AND rw.session = 'morning';
INSERT INTO appointment_room_date_capacity
    (id, room_id, service_date, session, total_capacity, occupied_capacity,
     room_window_id, room_window_version, version, created_at, updated_at)
SELECT '41000000-0000-4000-8000-000000000005', @room_dr101, @yesterday, 'morning', 4, 1,
       rw.id, rw.version, 1, @now, @now
FROM appointment_room_weekly_windows rw
WHERE rw.room_id = @room_dr101 AND rw.weekday = WEEKDAY(@yesterday) + 1 AND rw.session = 'morning';

SET @report_draft = '50000000-0000-4000-8000-000000000001';
SET @report_published = '50000000-0000-4000-8000-000000000002';
SET @report_corrected = '50000000-0000-4000-8000-000000000003';
SET @version_draft = '51000000-0000-4000-8000-000000000001';
SET @version_published = '51000000-0000-4000-8000-000000000002';
SET @version_initial = '51000000-0000-4000-8000-000000000003';
SET @version_correction = '51000000-0000-4000-8000-000000000004';

INSERT INTO appointment_examination_reports
    (id, booking_id, patient_account_id, patient_display_name_snapshot, patient_phone_masked_snapshot,
     department_id, department_name_snapshot, item_id, item_name_snapshot, room_id,
     campus_id_snapshot, campus_name_snapshot, building_snapshot, floor_number_snapshot,
     room_number_snapshot, status, performed_by, performed_by_display_name_snapshot,
     examination_started_at, examination_completed_at, current_version_id,
     version, created_at, updated_at)
VALUES
(@report_draft, @booking_in_progress, @account_patient_4, '赵强', '139****0004', @dept_radiology_main, '放射科', @item_urgent_ct, '当日急诊CT', @room_ct201, @campus_main, '总院区', '影像楼', 2, 'CT201', 'draft', @account_doctor_1, '陈医生', IF(TIMESTAMPDIFF(SECOND, @current_session_started_at_utc, DATE_SUB(@now, INTERVAL 20 MINUTE)) > 0, DATE_SUB(@now, INTERVAL 20 MINUTE), CAST(@current_session_started_at_utc AS DATETIME)), NULL, NULL, 1, IF(TIMESTAMPDIFF(SECOND, @current_session_started_at_utc, DATE_SUB(@now, INTERVAL 20 MINUTE)) > 0, DATE_SUB(@now, INTERVAL 20 MINUTE), CAST(@current_session_started_at_utc AS DATETIME)), @now),
(@report_published, @booking_completed, @account_patient_1, '体验管理员', @phone_patient_1_masked, @dept_ultrasound_main, '超声科', @item_ultrasound, '腹部彩超', @room_us301, @campus_main, '总院区', '门诊楼', 3, 'US301', 'published', @account_doctor_2, '周医生', CONVERT_TZ(TIMESTAMP(@yesterday, '09:10:00'), '+08:00', '+00:00'), CONVERT_TZ(TIMESTAMP(@yesterday, '09:50:00'), '+08:00', '+00:00'), NULL, 1, CONVERT_TZ(TIMESTAMP(@yesterday, '09:10:00'), '+08:00', '+00:00'), @now),
(@report_corrected, @booking_corrected, @account_patient_1, '体验管理员', @phone_patient_1_masked, @dept_radiology_main, '放射科', @item_xray, '胸部X线正侧位', @room_dr101, @campus_main, '总院区', '影像楼', 1, 'DR101', 'published', @account_doctor_1, '陈医生', CONVERT_TZ(TIMESTAMP(@two_days_ago, '09:20:00'), '+08:00', '+00:00'), CONVERT_TZ(TIMESTAMP(@two_days_ago, '10:10:00'), '+08:00', '+00:00'), NULL, 2, CONVERT_TZ(TIMESTAMP(@two_days_ago, '09:20:00'), '+08:00', '+00:00'), @now);

INSERT INTO appointment_examination_report_versions
    (id, report_id, version_no, version_kind, status, objective_findings, impression,
     recommendation, notes, correction_reason, authored_by, authored_by_display_name_snapshot,
     published_by, published_by_display_name_snapshot, published_at, created_at, updated_at)
VALUES
(@version_draft, @report_draft, 1, 'initial', 'draft', '双肺纹理稍增多，未见明显实变影。', '待结合完整序列形成结论。', '', '测试中的报告草稿', NULL, @account_doctor_1, '陈医生', NULL, NULL, NULL, @now, @now),
(@version_published, @report_published, 1, 'initial', 'published', '肝脏形态大小正常，胆囊壁光滑，胰脾双肾未见明显异常。', '腹部超声未见明显异常。', '如有不适请结合临床随诊。', '', NULL, @account_doctor_2, '周医生', @account_doctor_2, '周医生', CONVERT_TZ(TIMESTAMP(@yesterday, '10:00:00'), '+08:00', '+00:00'), CONVERT_TZ(TIMESTAMP(@yesterday, '09:10:00'), '+08:00', '+00:00'), CONVERT_TZ(TIMESTAMP(@yesterday, '10:00:00'), '+08:00', '+00:00')),
(@version_initial, @report_corrected, 1, 'initial', 'superseded', '双肺纹理清晰，右下肺见小片状高密度影。', '右下肺炎症可能。', '建议抗炎治疗后复查。', '', NULL, @account_doctor_1, '陈医生', @account_doctor_1, '陈医生', DATE_SUB(@now, INTERVAL 2 DAY), DATE_SUB(@now, INTERVAL 2 DAY), DATE_SUB(@now, INTERVAL 1 DAY)),
(@version_correction, @report_corrected, 2, 'correction', 'published', '双肺纹理清晰，左下肺见小片状高密度影。', '左下肺炎症可能。', '建议抗炎治疗后复查。', '已复核原始影像。', '原报告左右侧录入错误', @account_doctor_1, '陈医生', @account_doctor_1, '陈医生', DATE_SUB(@now, INTERVAL 1 DAY), DATE_SUB(@now, INTERVAL 1 DAY), @now);

UPDATE appointment_examination_reports SET current_version_id = @version_draft WHERE id = @report_draft;
UPDATE appointment_examination_reports SET current_version_id = @version_published WHERE id = @report_published;
UPDATE appointment_examination_reports SET current_version_id = @version_correction WHERE id = @report_corrected;

-- 保留一条本人已读记录，消息页能够同时展示已读和未读；工作人员消息默认全部未读。
INSERT INTO appointment_message_reads (account_id, message_key, read_at)
VALUES (@account_patient_1, CONCAT('booking:', @booking_completed, ':patient:created'), DATE_SUB(@now, INTERVAL 12 HOUR));
