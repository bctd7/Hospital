-- 组织变更审计：operation_id 同时承担组织写请求的幂等键。
-- 单独建表，避免把组织单元 ID 塞进只允许账号 ID 的 identity_authorization_audit。
CREATE TABLE identity_organization_audit (
    id                  CHAR(36)     NOT NULL,
    operation_id        CHAR(36)     NOT NULL,
    operator_account_id CHAR(36)     NOT NULL,
    unit_id             CHAR(36)     NOT NULL,
    action              VARCHAR(128) NOT NULL,
    before_data         JSON         NULL,
    after_data          JSON         NOT NULL,
    request_id          VARCHAR(64)  NULL,
    created_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_identity_organization_audit_operation (operation_id),
    KEY idx_identity_organization_audit_unit (unit_id, created_at),
    KEY idx_identity_organization_audit_operator (operator_account_id, created_at),
    CONSTRAINT fk_identity_organization_audit_operator
        FOREIGN KEY (operator_account_id)
            REFERENCES identity_accounts (id),
    CONSTRAINT fk_identity_organization_audit_unit
        FOREIGN KEY (unit_id)
            REFERENCES identity_organization_units (id)
) ENGINE=InnoDB;
