CREATE TABLE appointment_examination_items (
    id                  CHAR(36)      NOT NULL,
    owner_department_id CHAR(36)      NOT NULL,
    name                VARCHAR(128)  NOT NULL,
    description         TEXT          NOT NULL,
    estimated_duration_minutes SMALLINT UNSIGNED NOT NULL,
    report_template_objective_findings TEXT NOT NULL,
    report_template_impression          TEXT NOT NULL,
    report_template_recommendation      TEXT NOT NULL,
    report_template_notes               TEXT NOT NULL,
    report_template_version             BIGINT UNSIGNED NOT NULL DEFAULT 0,
    status              VARCHAR(16)   NOT NULL,
    version             BIGINT UNSIGNED NOT NULL,
    created_at          DATETIME(3)   NOT NULL,
    updated_at          DATETIME(3)   NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_examination_items_department_name
        (owner_department_id, name),
    KEY idx_appointment_examination_items_department_status
        (owner_department_id, status, name, id),
    CONSTRAINT chk_appointment_examination_items_status
        CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_appointment_examination_items_version
        CHECK (version > 0),
    CONSTRAINT chk_appointment_examination_items_duration
        CHECK (estimated_duration_minutes BETWEEN 5 AND 480 AND MOD(estimated_duration_minutes, 5) = 0),
    CONSTRAINT chk_appointment_examination_items_report_template_version
        CHECK (report_template_version >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_examination_item_operations (
    operation_id        CHAR(36)      NOT NULL,
    operator_account_id CHAR(36)      NOT NULL,
    item_id             CHAR(36)      NOT NULL,
    action              VARCHAR(128)  NOT NULL,
    request_fingerprint CHAR(64)      NOT NULL,
    result_data         JSON          NOT NULL,
    created_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (operation_id),
    KEY idx_appointment_examination_item_operations_item (item_id, created_at),
    CONSTRAINT fk_appointment_examination_item_operations_item
        FOREIGN KEY (item_id)
            REFERENCES appointment_examination_items (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_examination_item_audit (
    id                  CHAR(36)      NOT NULL,
    operation_id        CHAR(36)      NOT NULL,
    operator_account_id CHAR(36)      NOT NULL,
    item_id             CHAR(36)      NOT NULL,
    action              VARCHAR(128)  NOT NULL,
    before_data         JSON          NULL,
    after_data          JSON          NOT NULL,
    request_id          VARCHAR(64)   NULL,
    created_at          DATETIME(3)   NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_examination_item_audit_operation (operation_id),
    KEY idx_appointment_examination_item_audit_item (item_id, created_at),
    KEY idx_appointment_examination_item_audit_operator (operator_account_id, created_at),
    CONSTRAINT fk_appointment_examination_item_audit_item
        FOREIGN KEY (item_id)
            REFERENCES appointment_examination_items (id),
    CONSTRAINT fk_appointment_examination_item_audit_operation
        FOREIGN KEY (operation_id)
            REFERENCES appointment_examination_item_operations (operation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_examination_item_configuration_transactions (
    transaction_id             CHAR(36)      NOT NULL,
    operation_id               CHAR(36)      NOT NULL,
    operator_account_id        CHAR(36)      NOT NULL,
    action                     VARCHAR(16)   NOT NULL,
    item_id                    CHAR(36)      NOT NULL,
    owner_department_id        CHAR(36)      NOT NULL,
    item_name                  VARCHAR(128)  NOT NULL,
    description_snapshot       TEXT          NOT NULL,
    estimated_duration_minutes SMALLINT UNSIGNED NOT NULL,
    expected_version           BIGINT UNSIGNED NOT NULL,
    state                      VARCHAR(16)   NOT NULL,
    request_fingerprint        CHAR(64)      NOT NULL,
    result_version             BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at                 DATETIME(3)   NOT NULL,
    updated_at                 DATETIME(3)   NOT NULL,
    PRIMARY KEY (transaction_id),
    UNIQUE KEY uq_appointment_item_configuration_operation (operation_id),
    KEY idx_appointment_item_configuration_item (item_id, created_at),
    KEY idx_appointment_item_configuration_state (state, updated_at),
    CONSTRAINT chk_appointment_item_configuration_action CHECK (action IN ('create', 'update')),
    CONSTRAINT chk_appointment_item_configuration_state CHECK (state IN ('prepared', 'confirmed', 'cancelled')),
    CONSTRAINT chk_appointment_item_configuration_duration CHECK (
        estimated_duration_minutes BETWEEN 5 AND 480 AND MOD(estimated_duration_minutes, 5) = 0
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_rooms (
    id              CHAR(36)        NOT NULL,
    department_id   CHAR(36)        NOT NULL,
    campus_id       CHAR(36)        NOT NULL,
    building        VARCHAR(64)     NOT NULL,
    floor_number    SMALLINT        NOT NULL,
    room_number     VARCHAR(32)     NOT NULL,
    retired_at      DATETIME(3)     NULL,
    version         BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME(3)     NOT NULL,
    updated_at      DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_rooms_location (campus_id, building, floor_number, room_number),
    KEY idx_appointment_rooms_department_active_location
        (department_id, retired_at, building, floor_number, room_number, id),
    CONSTRAINT chk_appointment_rooms_campus_id CHECK (
        campus_id REGEXP '^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$'
    ),
    CONSTRAINT chk_appointment_rooms_building CHECK (CHAR_LENGTH(TRIM(building)) BETWEEN 1 AND 64),
    CONSTRAINT chk_appointment_rooms_floor CHECK (floor_number BETWEEN -9 AND 99 AND floor_number <> 0),
    CONSTRAINT chk_appointment_rooms_room_number CHECK (
        CHAR_LENGTH(TRIM(room_number)) BETWEEN 1 AND 32
        AND room_number REGEXP '^[[:alnum:]_-]+$'
    ),
    CONSTRAINT chk_appointment_rooms_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_room_examination_items (
    id              CHAR(36)        NOT NULL,
    room_id         CHAR(36)        NOT NULL,
    item_id         CHAR(36)        NOT NULL,
    status          VARCHAR(16)     NOT NULL,
    version         BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME(3)     NOT NULL,
    updated_at      DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_room_examination_items_pair (room_id, item_id),
    KEY idx_appointment_room_examination_items_item_status (item_id, status, room_id),
    CONSTRAINT fk_appointment_room_examination_items_room
        FOREIGN KEY (room_id) REFERENCES appointment_rooms (id),
    CONSTRAINT fk_appointment_room_examination_items_item
        FOREIGN KEY (item_id) REFERENCES appointment_examination_items (id),
    CONSTRAINT chk_appointment_room_examination_items_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_appointment_room_examination_items_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_room_weekly_windows (
    id                  CHAR(36)        NOT NULL,
    room_id             CHAR(36)        NOT NULL,
    weekday             TINYINT UNSIGNED NOT NULL,
    session             VARCHAR(16)     NOT NULL,
    open_time           TIME            NOT NULL,
    close_time          TIME            NOT NULL,
    active_capacity     INT UNSIGNED    NOT NULL,
    status              VARCHAR(16)     NOT NULL,
    version             BIGINT UNSIGNED NOT NULL,
    created_at          DATETIME(3)     NOT NULL,
    updated_at          DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_room_weekly_windows_slot (room_id, weekday, session),
    KEY idx_appointment_room_weekly_windows_status (room_id, status, weekday, session),
    CONSTRAINT fk_appointment_room_weekly_windows_room
        FOREIGN KEY (room_id) REFERENCES appointment_rooms (id),
    CONSTRAINT chk_appointment_room_weekly_windows_weekday CHECK (weekday BETWEEN 1 AND 7),
    CONSTRAINT chk_appointment_room_weekly_windows_session CHECK (session IN ('morning', 'afternoon')),
    CONSTRAINT chk_appointment_room_weekly_windows_time CHECK (open_time < close_time),
    CONSTRAINT chk_appointment_room_weekly_windows_session_time
        CHECK ((session = 'morning' AND close_time <= '12:00:00') OR
               (session = 'afternoon' AND open_time >= '12:00:00')),
    CONSTRAINT chk_appointment_room_weekly_windows_capacity CHECK (active_capacity > 0),
    CONSTRAINT chk_appointment_room_weekly_windows_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_appointment_room_weekly_windows_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_item_weekly_windows (
    id                      CHAR(36)        NOT NULL,
    item_id                 CHAR(36)        NOT NULL,
    weekday                 TINYINT UNSIGNED NOT NULL,
    session                 VARCHAR(16)     NOT NULL,
    start_time              TIME            NOT NULL,
    booking_cutoff_time     TIME            NOT NULL,
    end_time                TIME            NOT NULL,
    status                  VARCHAR(16)     NOT NULL,
    version                 BIGINT UNSIGNED NOT NULL,
    created_at              DATETIME(3)     NOT NULL,
    updated_at              DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_item_weekly_windows_slot (item_id, weekday, session),
    KEY idx_appointment_item_weekly_windows_status (item_id, status, weekday, session),
    CONSTRAINT fk_appointment_item_weekly_windows_item
        FOREIGN KEY (item_id) REFERENCES appointment_examination_items (id),
    CONSTRAINT chk_appointment_item_weekly_windows_weekday CHECK (weekday BETWEEN 1 AND 7),
    CONSTRAINT chk_appointment_item_weekly_windows_session CHECK (session IN ('morning', 'afternoon')),
    CONSTRAINT chk_appointment_item_weekly_windows_time
        CHECK (start_time <= booking_cutoff_time AND booking_cutoff_time < end_time),
    CONSTRAINT chk_appointment_item_weekly_windows_session_time
        CHECK ((session = 'morning' AND end_time <= '12:00:00') OR
               (session = 'afternoon' AND start_time >= '12:00:00')),
    CONSTRAINT chk_appointment_item_weekly_windows_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_appointment_item_weekly_windows_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_resource_operations (
    operation_id        CHAR(36)        NOT NULL,
    operator_account_id CHAR(36)        NOT NULL,
    resource_type       VARCHAR(32)     NOT NULL,
    resource_id         CHAR(36)        NOT NULL,
    action              VARCHAR(128)    NOT NULL,
    request_fingerprint CHAR(64)        NOT NULL,
    result_data         JSON            NOT NULL,
    created_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (operation_id),
    KEY idx_appointment_resource_operations_resource (resource_type, resource_id, created_at),
    KEY idx_appointment_resource_operations_operator (operator_account_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_resource_audit (
    id                  CHAR(36)        NOT NULL,
    operation_id        CHAR(36)        NOT NULL,
    operator_account_id CHAR(36)        NOT NULL,
    department_id       CHAR(36)        NOT NULL,
    resource_type       VARCHAR(32)     NOT NULL,
    resource_id         CHAR(36)        NOT NULL,
    action              VARCHAR(128)    NOT NULL,
    before_data         JSON            NULL,
    after_data          JSON            NOT NULL,
    request_id          VARCHAR(64)     NULL,
    created_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_resource_audit_operation (operation_id),
    KEY idx_appointment_resource_audit_department (department_id, created_at),
    KEY idx_appointment_resource_audit_resource (resource_type, resource_id, created_at),
    CONSTRAINT fk_appointment_resource_audit_operation
        FOREIGN KEY (operation_id) REFERENCES appointment_resource_operations (operation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_room_date_capacity (
    id                  CHAR(36)        NOT NULL,
    room_id             CHAR(36)        NOT NULL,
    service_date        DATE            NOT NULL,
    session             VARCHAR(16)     NOT NULL,
    total_capacity      INT UNSIGNED    NOT NULL,
    occupied_capacity   INT UNSIGNED    NOT NULL DEFAULT 0,
    room_window_id      CHAR(36)        NOT NULL,
    room_window_version BIGINT UNSIGNED NOT NULL,
    version             BIGINT UNSIGNED NOT NULL,
    created_at          DATETIME(3)     NOT NULL,
    updated_at          DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_room_date_capacity_slot (room_id, service_date, session),
    KEY idx_appointment_room_date_capacity_date (service_date, room_id, session),
    CONSTRAINT fk_appointment_room_date_capacity_room
        FOREIGN KEY (room_id) REFERENCES appointment_rooms (id),
    CONSTRAINT fk_appointment_room_date_capacity_window
        FOREIGN KEY (room_window_id) REFERENCES appointment_room_weekly_windows (id),
    CONSTRAINT chk_appointment_room_date_capacity_session
        CHECK (session IN ('morning', 'afternoon')),
    CONSTRAINT chk_appointment_room_date_capacity_total CHECK (total_capacity > 0),
    CONSTRAINT chk_appointment_room_date_capacity_occupied
        CHECK (occupied_capacity <= total_capacity),
    CONSTRAINT chk_appointment_room_date_capacity_window_version
        CHECK (room_window_version > 0),
    CONSTRAINT chk_appointment_room_date_capacity_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 每次成功预约永久占用患者当周一次额度；取消、完成和失约均不递减。
-- 使用周一日期作为分区键后无需定时重置，进入下一周自然创建新记录。
CREATE TABLE appointment_patient_weekly_quota_usage (
    patient_account_id CHAR(36)       NOT NULL,
    week_start_date    DATE           NOT NULL,
    used_count         INT UNSIGNED   NOT NULL,
    version            BIGINT UNSIGNED NOT NULL,
    created_at         DATETIME(3)    NOT NULL,
    updated_at         DATETIME(3)    NOT NULL,
    PRIMARY KEY (patient_account_id, week_start_date),
    CONSTRAINT chk_appointment_patient_weekly_quota_usage_count CHECK (used_count >= 0),
    CONSTRAINT chk_appointment_patient_weekly_quota_usage_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_bookings (
    id                           CHAR(36)        NOT NULL,
    patient_account_id           CHAR(36)        NOT NULL,
    patient_display_name_snapshot VARCHAR(128)   NOT NULL,
    patient_phone_masked_snapshot VARCHAR(32)    NOT NULL,
    patient_phone_last4_snapshot  CHAR(4)        NOT NULL,
    department_id                CHAR(36)        NOT NULL,
    item_id                      CHAR(36)        NOT NULL,
    room_id                      CHAR(36)        NOT NULL,
    service_date                 DATE            NOT NULL,
    session                      VARCHAR(16)     NOT NULL,
    status                       VARCHAR(16)     NOT NULL,
    room_open_time_snapshot      TIME            NOT NULL,
    room_close_time_snapshot     TIME            NOT NULL,
    item_start_time_snapshot     TIME            NOT NULL,
    item_end_time_snapshot       TIME            NOT NULL,
    item_cutoff_time_snapshot    TIME            NOT NULL,
    estimated_duration_minutes_snapshot SMALLINT UNSIGNED NOT NULL,
    started_at                   DATETIME(3)      NULL,
    started_by                   CHAR(36)         NULL,
    started_by_display_name_snapshot VARCHAR(128) NULL,
    examination_ended_at           DATETIME(3)      NULL,
    examination_ended_by           CHAR(36)         NULL,
    examination_ended_by_display_name_snapshot VARCHAR(128) NULL,
    completed_at                 DATETIME(3)      NULL,
    completed_by                 CHAR(36)         NULL,
    completed_by_display_name_snapshot VARCHAR(128) NULL,
    version                      BIGINT UNSIGNED  NOT NULL,
    created_at                   DATETIME(3)      NOT NULL,
    updated_at                   DATETIME(3)      NOT NULL,
    PRIMARY KEY (id),
    KEY idx_appointment_bookings_patient
        (patient_account_id, service_date DESC, created_at DESC, id),
    KEY idx_appointment_bookings_department
        (department_id, service_date, session, created_at, id),
    KEY idx_appointment_bookings_department_patient_name
        (department_id, patient_display_name_snapshot, service_date, id),
    KEY idx_appointment_bookings_department_patient_phone
        (department_id, patient_phone_last4_snapshot, service_date, id),
    KEY idx_appointment_bookings_room_capacity
        (room_id, service_date, session, status, created_at, id),
    KEY idx_appointment_bookings_cleanup
        (status, service_date, item_end_time_snapshot, id),
    CONSTRAINT fk_appointment_bookings_item
        FOREIGN KEY (item_id) REFERENCES appointment_examination_items (id),
    CONSTRAINT fk_appointment_bookings_room
        FOREIGN KEY (room_id) REFERENCES appointment_rooms (id),
    CONSTRAINT chk_appointment_bookings_session
        CHECK (session IN ('morning', 'afternoon')),
    CONSTRAINT chk_appointment_bookings_status
        CHECK (status IN ('confirmed', 'queued', 'called', 'in_progress', 'report_pending', 'completed', 'no_show', 'canceled')),
    CONSTRAINT chk_appointment_bookings_room_time
        CHECK (room_open_time_snapshot < room_close_time_snapshot),
    CONSTRAINT chk_appointment_bookings_item_time
        CHECK (
            item_start_time_snapshot <= item_cutoff_time_snapshot
            AND item_cutoff_time_snapshot < item_end_time_snapshot
            AND room_open_time_snapshot <= item_start_time_snapshot
            AND item_end_time_snapshot <= room_close_time_snapshot
        ),
    CONSTRAINT chk_appointment_bookings_duration
        CHECK (estimated_duration_minutes_snapshot BETWEEN 5 AND 480 AND MOD(estimated_duration_minutes_snapshot, 5) = 0),
    CONSTRAINT chk_appointment_bookings_lifecycle CHECK (
        (status IN ('confirmed', 'queued', 'called', 'no_show', 'canceled') AND started_at IS NULL AND started_by IS NULL
            AND examination_ended_at IS NULL AND examination_ended_by IS NULL AND completed_at IS NULL AND completed_by IS NULL)
        OR (status = 'in_progress' AND started_at IS NOT NULL AND started_by IS NOT NULL
            AND examination_ended_at IS NULL AND examination_ended_by IS NULL AND completed_at IS NULL AND completed_by IS NULL)
        OR (status = 'report_pending' AND started_at IS NOT NULL AND started_by IS NOT NULL
            AND examination_ended_at IS NOT NULL AND examination_ended_by IS NOT NULL
            AND completed_at IS NULL AND completed_by IS NULL)
        OR (status = 'completed' AND started_at IS NOT NULL AND started_by IS NOT NULL
            AND examination_ended_at IS NOT NULL AND examination_ended_by IS NOT NULL
            AND completed_at IS NOT NULL AND completed_by IS NOT NULL)
    ),
    CONSTRAINT chk_appointment_bookings_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_check_queues (
    id                        CHAR(36)        NOT NULL,
    room_id                   CHAR(36)        NOT NULL,
    service_date              DATE            NOT NULL,
    session                   VARCHAR(16)     NOT NULL,
    next_ticket_number        BIGINT UNSIGNED NOT NULL DEFAULT 1,
    call_sequence             BIGINT UNSIGNED NOT NULL DEFAULT 0,
    current_called_booking_id CHAR(36)        NULL,
    version                   BIGINT UNSIGNED NOT NULL DEFAULT 1,
    created_at                DATETIME(3)     NOT NULL,
    updated_at                DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_check_queues_scope (room_id, service_date, session),
    KEY idx_appointment_check_queues_current (current_called_booking_id),
    CONSTRAINT fk_appointment_check_queues_room FOREIGN KEY (room_id) REFERENCES appointment_rooms (id),
    CONSTRAINT chk_appointment_check_queues_session CHECK (session IN ('morning', 'afternoon')),
    CONSTRAINT chk_appointment_check_queues_numbers CHECK (next_ticket_number > 0 AND version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_check_queue_entries (
    booking_id                   CHAR(36)        NOT NULL,
    queue_id                     CHAR(36)        NOT NULL,
    ticket_number                BIGINT UNSIGNED NOT NULL,
    eligible_after_call_sequence BIGINT UNSIGNED NOT NULL DEFAULT 0,
    call_attempts                BIGINT UNSIGNED NOT NULL DEFAULT 0,
    checked_in_at                DATETIME(3)     NOT NULL,
    called_at                    DATETIME(3)     NULL,
    call_deadline                DATETIME(3)     NULL,
    updated_at                   DATETIME(3)     NOT NULL,
    PRIMARY KEY (booking_id),
    UNIQUE KEY uk_appointment_check_queue_entries_ticket (queue_id, ticket_number),
    KEY idx_appointment_check_queue_entries_next (queue_id, eligible_after_call_sequence, ticket_number),
    KEY idx_appointment_check_queue_entries_deadline (call_deadline, booking_id),
    CONSTRAINT fk_appointment_check_queue_entries_booking FOREIGN KEY (booking_id) REFERENCES appointment_bookings (id),
    CONSTRAINT fk_appointment_check_queue_entries_queue FOREIGN KEY (queue_id) REFERENCES appointment_check_queues (id),
    CONSTRAINT chk_appointment_check_queue_entries_ticket CHECK (ticket_number > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_check_queue_events (
    id            CHAR(36)        NOT NULL,
    queue_id      CHAR(36)        NOT NULL,
    booking_id    CHAR(36)        NOT NULL,
    event_type    VARCHAR(24)     NOT NULL,
    call_sequence BIGINT UNSIGNED NOT NULL DEFAULT 0,
    occurred_at   DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    KEY idx_appointment_check_queue_events_booking (booking_id, occurred_at, id),
    KEY idx_appointment_check_queue_events_queue (queue_id, occurred_at, id),
    CONSTRAINT fk_appointment_check_queue_events_queue FOREIGN KEY (queue_id) REFERENCES appointment_check_queues (id),
    CONSTRAINT fk_appointment_check_queue_events_booking FOREIGN KEY (booking_id) REFERENCES appointment_bookings (id),
    CONSTRAINT chk_appointment_check_queue_events_type CHECK (event_type IN ('checked_in', 'called', 'deferred', 'started', 'ended'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 消息正文由预约与报告事实动态生成；这里只保存每个账号独立的已读游标。
CREATE TABLE appointment_message_reads (
    account_id  CHAR(36)     NOT NULL,
    message_key VARCHAR(160) NOT NULL,
    read_at     DATETIME(3)  NOT NULL,
    PRIMARY KEY (account_id, message_key),
    KEY idx_appointment_message_reads_account_time (account_id, read_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- A patient may hold only one confirmed or in-progress booking for the same item in a concrete date/session.
-- The room is deliberately excluded so changing rooms cannot bypass duplicate protection.
-- The guard is released after completion, cancellation, or no-show.
CREATE TABLE appointment_patient_item_session_claims (
    patient_account_id CHAR(36)    NOT NULL,
    item_id            CHAR(36)    NOT NULL,
    service_date       DATE        NOT NULL,
    session            VARCHAR(16) NOT NULL,
    booking_id         CHAR(36)    NOT NULL,
    created_at         DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (patient_account_id, item_id, service_date, session),
    UNIQUE KEY uk_appointment_patient_item_session_claims_booking (booking_id),
    CONSTRAINT chk_appointment_patient_item_session_claims_session
        CHECK (session IN ('morning', 'afternoon'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE appointment_booking_operations (
    operation_id        CHAR(36)        NOT NULL,
    operator_account_id CHAR(36)        NOT NULL,
    booking_id          CHAR(36)        NOT NULL,
    action              VARCHAR(64)     NOT NULL,
    request_fingerprint CHAR(64)        NOT NULL,
    result_data         JSON            NOT NULL,
    created_at          DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (operation_id),
    KEY idx_appointment_booking_operations_booking (booking_id, created_at),
    KEY idx_appointment_booking_operations_operator (operator_account_id, created_at),
    CONSTRAINT chk_appointment_booking_operations_action CHECK (
        action IN ('create', 'delete_by_patient', 'delete_by_staff', 'delete_by_configuration',
                   'mark_no_show', 'check_in', 'call_next', 'start_examination', 'end_examination',
                   'save_report_draft', 'complete_and_publish_report', 'correct_report')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 一次预约只产生一份报告主体。检查项目和房间信息在首次建档时快照，
-- 避免后续目录改名或房间地址调整改变既有正式报告。
CREATE TABLE appointment_examination_reports (
    id                           CHAR(36)        NOT NULL,
    booking_id                   CHAR(36)        NOT NULL,
    patient_account_id           CHAR(36)        NOT NULL,
    patient_display_name_snapshot VARCHAR(128)   NOT NULL,
    patient_phone_masked_snapshot VARCHAR(32)    NOT NULL,
    department_id                CHAR(36)        NOT NULL,
    department_name_snapshot     VARCHAR(128)    NOT NULL,
    item_id                      CHAR(36)        NOT NULL,
    item_name_snapshot           VARCHAR(128)    NOT NULL,
    room_id                      CHAR(36)        NOT NULL,
    campus_id_snapshot           CHAR(36)        NOT NULL,
    campus_name_snapshot         VARCHAR(128)    NOT NULL,
    building_snapshot            VARCHAR(64)     NOT NULL,
    floor_number_snapshot        SMALLINT        NOT NULL,
    room_number_snapshot         VARCHAR(32)     NOT NULL,
    status                       VARCHAR(16)      NOT NULL,
    performed_by                 CHAR(36)        NULL,
    performed_by_display_name_snapshot VARCHAR(128) NULL,
    examination_started_at       DATETIME(3)      NULL,
    examination_completed_at     DATETIME(3)      NULL,
    current_version_id           CHAR(36)        NULL,
    version                      BIGINT UNSIGNED NOT NULL,
    created_at                   DATETIME(3)      NOT NULL,
    updated_at                   DATETIME(3)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_examination_reports_booking (booking_id),
    KEY idx_appointment_examination_reports_patient (patient_account_id, status, updated_at DESC, id),
    KEY idx_appointment_examination_reports_department (department_id, status, updated_at DESC, id),
    CONSTRAINT fk_appointment_examination_reports_booking
        FOREIGN KEY (booking_id) REFERENCES appointment_bookings (id),
    CONSTRAINT chk_appointment_examination_reports_status
        CHECK (status IN ('draft', 'published')),
    CONSTRAINT chk_appointment_examination_reports_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- 草稿版本允许原地编辑；正式版本发布后只读。更正始终新增版本，旧版仅改为 superseded。
CREATE TABLE appointment_examination_report_versions (
    id                  CHAR(36)        NOT NULL,
    report_id           CHAR(36)        NOT NULL,
    version_no          BIGINT UNSIGNED NOT NULL,
    version_kind        VARCHAR(16)     NOT NULL,
    status              VARCHAR(16)     NOT NULL,
    objective_findings  TEXT            NOT NULL,
    impression          TEXT            NOT NULL,
    recommendation      TEXT            NOT NULL,
    notes               TEXT            NOT NULL,
    correction_reason   VARCHAR(512)    NULL,
    authored_by         CHAR(36)        NOT NULL,
    authored_by_display_name_snapshot VARCHAR(128) NOT NULL,
    published_by        CHAR(36)        NULL,
    published_by_display_name_snapshot VARCHAR(128) NULL,
    published_at        DATETIME(3)      NULL,
    created_at          DATETIME(3)      NOT NULL,
    updated_at          DATETIME(3)      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_examination_report_versions_no (report_id, version_no),
    KEY idx_appointment_examination_report_versions_status (report_id, status, version_no DESC),
    CONSTRAINT fk_appointment_examination_report_versions_report
        FOREIGN KEY (report_id) REFERENCES appointment_examination_reports (id),
    CONSTRAINT chk_appointment_examination_report_versions_kind
        CHECK (version_kind IN ('initial', 'correction')),
    CONSTRAINT chk_appointment_examination_report_versions_status
        CHECK (status IN ('draft', 'published', 'superseded')),
    CONSTRAINT chk_appointment_examination_report_versions_publish CHECK (
        (status = 'draft' AND published_by IS NULL AND published_at IS NULL)
        OR (status IN ('published', 'superseded') AND published_by IS NOT NULL AND published_at IS NOT NULL)
    ),
    CONSTRAINT chk_appointment_examination_report_versions_correction CHECK (
        (version_kind = 'initial' AND correction_reason IS NULL)
        OR (version_kind = 'correction' AND CHAR_LENGTH(TRIM(correction_reason)) > 0)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

ALTER TABLE appointment_examination_reports
    ADD CONSTRAINT fk_appointment_examination_reports_current_version
        FOREIGN KEY (current_version_id)
        REFERENCES appointment_examination_report_versions (id);
