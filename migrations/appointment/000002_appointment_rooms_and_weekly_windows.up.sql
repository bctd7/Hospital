CREATE TABLE appointment_rooms (
    id              CHAR(36)        NOT NULL,
    department_id   CHAR(36)        NOT NULL,
    name            VARCHAR(128)    NOT NULL,
    status          VARCHAR(16)     NOT NULL,
    version         BIGINT UNSIGNED NOT NULL,
    created_at      DATETIME(3)     NOT NULL,
    updated_at      DATETIME(3)     NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_appointment_rooms_department_name (department_id, name),
    KEY idx_appointment_rooms_department_status (department_id, status, name, id),
    CONSTRAINT chk_appointment_rooms_status CHECK (status IN ('active', 'disabled')),
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
