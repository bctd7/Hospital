CREATE TABLE guidance_precedence_graph_locks (
    graph_name VARCHAR(64) NOT NULL,
    PRIMARY KEY (graph_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO guidance_precedence_graph_locks (graph_name) VALUES ('examination_item_precedence');

CREATE TABLE guidance_precedence_rules (
    id                          CHAR(36)     NOT NULL,
    owner_item_id               CHAR(36)     NOT NULL,
    predecessor_item_id         CHAR(36)     NOT NULL,
    predecessor_department_id   CHAR(36)     NOT NULL,
    predecessor_item_name       VARCHAR(128) NOT NULL,
    successor_item_id           CHAR(36)     NOT NULL,
    successor_department_id     CHAR(36)     NOT NULL,
    successor_item_name         VARCHAR(128) NOT NULL,
    staff_reason                VARCHAR(512) NOT NULL,
    patient_message             VARCHAR(512) NOT NULL DEFAULT '',
    created_by                  CHAR(36)     NOT NULL,
    create_operation_id         CHAR(36)     NOT NULL,
    version                     BIGINT       NOT NULL DEFAULT 1,
    created_at                  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at                  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uq_guidance_precedence_pair (predecessor_item_id, successor_item_id),
    UNIQUE KEY uq_guidance_precedence_create_operation (create_operation_id),
    KEY idx_guidance_precedence_predecessor (predecessor_item_id),
    KEY idx_guidance_precedence_successor (successor_item_id),
    KEY idx_guidance_precedence_owner (owner_item_id),
    KEY idx_guidance_precedence_successor_department (successor_department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE guidance_item_configurations (
    item_id                    CHAR(36)     NOT NULL,
    description                TEXT         NOT NULL,
    preparation_rules          JSON         NOT NULL,
    reminders                  JSON         NOT NULL,
    version                    BIGINT UNSIGNED NOT NULL,
    updated_by                 CHAR(36)     NOT NULL,
    created_at                 DATETIME(3)  NOT NULL,
    updated_at                 DATETIME(3)  NOT NULL,
    PRIMARY KEY (item_id),
    CONSTRAINT chk_guidance_item_configuration_version CHECK (version > 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE guidance_configuration_transactions (
    transaction_id             CHAR(36)     NOT NULL,
    operation_id               CHAR(36)     NOT NULL,
    action                     VARCHAR(16)  NOT NULL,
    item_id                    CHAR(36)     NOT NULL,
    operator_account_id        CHAR(36)     NOT NULL,
    state                      VARCHAR(16)  NOT NULL,
    request_fingerprint        CHAR(64)     NOT NULL,
    payload                    JSON         NOT NULL,
    before_configuration       JSON         NULL,
    before_precedence_rules    JSON         NULL,
    last_error                 VARCHAR(1024) NOT NULL DEFAULT '',
    retry_count                INT UNSIGNED NOT NULL DEFAULT 0,
    created_at                 DATETIME(3)  NOT NULL,
    updated_at                 DATETIME(3)  NOT NULL,
    PRIMARY KEY (transaction_id),
    UNIQUE KEY uq_guidance_configuration_operation (operation_id),
    KEY idx_guidance_configuration_state_updated (state, updated_at),
    KEY idx_guidance_configuration_item (item_id, created_at),
    CONSTRAINT chk_guidance_configuration_action CHECK (action IN ('create', 'update')),
    CONSTRAINT chk_guidance_configuration_state CHECK (
        state IN ('trying', 'prepared', 'confirming', 'confirmed', 'cancelling', 'cancelled')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE guidance_smart_appointment_plans (
    plan_id             CHAR(36)     NOT NULL,
    patient_account_id  CHAR(36)     NOT NULL,
    request_fingerprint CHAR(64)     NOT NULL,
    plan_payload        JSON         NOT NULL,
    expires_at          DATETIME(3)  NOT NULL,
    confirmed_booking_ids JSON       NOT NULL DEFAULT (JSON_ARRAY()),
    confirmed_at        DATETIME(3)  NULL,
    created_at          DATETIME(3)  NOT NULL,
    PRIMARY KEY (plan_id),
    KEY idx_guidance_smart_plan_patient (patient_account_id, expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
