CREATE TABLE guidance_precedence_graph_locks (
    graph_name VARCHAR(64) NOT NULL,
    PRIMARY KEY (graph_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO guidance_precedence_graph_locks (graph_name) VALUES ('examination_item_precedence');

CREATE TABLE guidance_precedence_rules (
    id                          CHAR(36)     NOT NULL,
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
    KEY idx_guidance_precedence_successor_department (successor_department_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
