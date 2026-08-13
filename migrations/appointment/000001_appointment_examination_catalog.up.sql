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
