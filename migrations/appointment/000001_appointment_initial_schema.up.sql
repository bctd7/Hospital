CREATE TABLE appointment_examination_items (
    id                  CHAR(36)      NOT NULL,
    owner_department_id CHAR(36)      NOT NULL,
    name                VARCHAR(128)  NOT NULL,
    description         TEXT          NOT NULL,
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
        CHECK (version > 0)
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

CREATE TABLE appointment_bookings (
    id                           CHAR(36)        NOT NULL,
    patient_account_id           CHAR(36)        NOT NULL,
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
    checked_in_at                DATETIME(3)      NULL,
    checked_in_by                CHAR(36)         NULL,
    version                      BIGINT UNSIGNED  NOT NULL,
    created_at                   DATETIME(3)      NOT NULL,
    updated_at                   DATETIME(3)      NOT NULL,
    PRIMARY KEY (id),
    KEY idx_appointment_bookings_patient
        (patient_account_id, service_date DESC, created_at DESC, id),
    KEY idx_appointment_bookings_department
        (department_id, service_date, session, created_at, id),
    KEY idx_appointment_bookings_room_capacity
        (room_id, service_date, session, status, created_at, id),
    KEY idx_appointment_bookings_cleanup
        (status, service_date, id),
    CONSTRAINT fk_appointment_bookings_item
        FOREIGN KEY (item_id) REFERENCES appointment_examination_items (id),
    CONSTRAINT fk_appointment_bookings_room
        FOREIGN KEY (room_id) REFERENCES appointment_rooms (id),
    CONSTRAINT chk_appointment_bookings_session
        CHECK (session IN ('morning', 'afternoon')),
    CONSTRAINT chk_appointment_bookings_status
        CHECK (status IN ('confirmed', 'checked_in')),
    CONSTRAINT chk_appointment_bookings_room_time
        CHECK (room_open_time_snapshot < room_close_time_snapshot),
    CONSTRAINT chk_appointment_bookings_item_time
        CHECK (
            item_start_time_snapshot <= item_cutoff_time_snapshot
            AND item_cutoff_time_snapshot < item_end_time_snapshot
            AND room_open_time_snapshot <= item_start_time_snapshot
            AND item_end_time_snapshot <= room_close_time_snapshot
        ),
    CONSTRAINT chk_appointment_bookings_check_in CHECK (
        (status = 'confirmed' AND checked_in_at IS NULL AND checked_in_by IS NULL)
        OR (status = 'checked_in' AND checked_in_at IS NOT NULL AND checked_in_by IS NOT NULL)
    ),
    CONSTRAINT chk_appointment_bookings_version CHECK (version > 0)
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
                   'delete_by_cleanup', 'check_in')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
