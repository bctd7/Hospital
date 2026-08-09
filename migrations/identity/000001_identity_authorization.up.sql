USE hospital_identity;

-- 平台账号主表：只保存所有账号共有的身份状态，不保存微信、手机号、科室或角色明细。
CREATE TABLE identity_accounts (
    id                    CHAR(36)     NOT NULL,
    account_type          VARCHAR(32)  NOT NULL,
    status                VARCHAR(16)  NOT NULL DEFAULT 'active',
    authorization_version BIGINT       NOT NULL DEFAULT 1,
    created_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at            DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    CONSTRAINT chk_identity_accounts_type CHECK (account_type IN ('patient', 'staff', 'service')),
    CONSTRAINT chk_identity_accounts_status CHECK (status IN ('active', 'disabled'))
) ENGINE=InnoDB;

-- 科室主数据：供工作人员当前归属和后续业务数据归属共同引用。
CREATE TABLE identity_departments (
    id          CHAR(36)     NOT NULL,
    parent_id   CHAR(36)     NULL,
    code        VARCHAR(64)  NOT NULL,
    name        VARCHAR(128) NOT NULL,
    status      VARCHAR(16)  NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_identity_departments_code (code),
    KEY idx_identity_departments_parent (parent_id),
    CONSTRAINT fk_identity_departments_parent FOREIGN KEY (parent_id) REFERENCES identity_departments (id),
    CONSTRAINT chk_identity_departments_status CHECK (status IN ('active', 'disabled'))
) ENGINE=InnoDB;

-- 工作人员扩展档案：仅 staff 账号存在；当前一个工作人员只属于一个科室。
CREATE TABLE identity_staff_profiles (
    account_id    CHAR(36)    NOT NULL,
    department_id CHAR(36)    NOT NULL,
    staff_no      VARCHAR(64)  NULL,
    created_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at    DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (account_id),
    KEY idx_identity_staff_department (department_id),
    CONSTRAINT fk_identity_staff_account FOREIGN KEY (account_id) REFERENCES identity_accounts (id),
    CONSTRAINT fk_identity_staff_department FOREIGN KEY (department_id) REFERENCES identity_departments (id)
) ENGINE=InnoDB;

-- 角色定义：描述“超级管理员”“部门医生”等稳定身份集合。
CREATE TABLE identity_roles (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code        VARCHAR(64)     NOT NULL,
    name        VARCHAR(128)    NOT NULL,
    status      VARCHAR(16)     NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_identity_roles_code (code),
    CONSTRAINT chk_identity_roles_status CHECK (status IN ('active', 'disabled'))
) ENGINE=InnoDB;

-- 权限定义：描述可被业务服务本地判断的具体操作能力。
CREATE TABLE identity_permissions (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    code        VARCHAR(128)    NOT NULL,
    service     VARCHAR(64)     NOT NULL,
    name        VARCHAR(128)    NOT NULL,
    status      VARCHAR(16)     NOT NULL DEFAULT 'active',
    created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_identity_permissions_code (code),
    KEY idx_identity_permissions_service (service),
    CONSTRAINT chk_identity_permissions_status CHECK (status IN ('active', 'disabled'))
) ENGINE=InnoDB;

-- 角色与权限的多对多关系：一个角色可有多个权限，一个权限可被多个角色复用。
CREATE TABLE identity_role_permissions (
    role_id       BIGINT UNSIGNED NOT NULL,
    permission_id BIGINT UNSIGNED NOT NULL,
    created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_identity_role_permissions_role FOREIGN KEY (role_id) REFERENCES identity_roles (id),
    CONSTRAINT fk_identity_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES identity_permissions (id)
) ENGINE=InnoDB;

-- 账号当前角色：account_id 是主键，因此首期明确限制一个账号最多一个角色。
CREATE TABLE identity_account_roles (
    account_id CHAR(36)        NOT NULL,
    role_id    BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (account_id),
    KEY idx_identity_account_roles_role (role_id),
    CONSTRAINT fk_identity_account_roles_account FOREIGN KEY (account_id) REFERENCES identity_accounts (id),
    CONSTRAINT fk_identity_account_roles_role FOREIGN KEY (role_id) REFERENCES identity_roles (id)
) ENGINE=InnoDB;

-- 授权审计：永久记录谁在何时对哪个账号执行了什么权限或组织变更。
CREATE TABLE identity_authorization_audit (
    id                  CHAR(36)     NOT NULL,
    operation_id        CHAR(36)     NOT NULL,
    operator_account_id CHAR(36)     NOT NULL,
    target_account_id   CHAR(36)     NOT NULL,
    action              VARCHAR(128) NOT NULL,
    before_data         JSON         NULL,
    after_data          JSON         NULL,
    request_id          VARCHAR(64)  NULL,
    created_at          DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_identity_authorization_audit_operation (operation_id),
    KEY idx_identity_authorization_audit_target (target_account_id, created_at),
    KEY idx_identity_authorization_audit_operator (operator_account_id, created_at),
    CONSTRAINT fk_identity_authorization_audit_operator FOREIGN KEY (operator_account_id) REFERENCES identity_accounts (id),
    CONSTRAINT fk_identity_authorization_audit_target FOREIGN KEY (target_account_id) REFERENCES identity_accounts (id)
) ENGINE=InnoDB;

-- 事务 Outbox：与授权变更同事务落库，供未来 Kafka 发布器可靠发送事件；不是用户资料表。
CREATE TABLE identity_outbox_events (
    event_id       CHAR(36)     NOT NULL,
    aggregate_id  CHAR(36)     NOT NULL,
    event_type     VARCHAR(128) NOT NULL,
    schema_version INT UNSIGNED NOT NULL,
    payload        JSON         NOT NULL,
    occurred_at    DATETIME(3)  NOT NULL,
    published_at   DATETIME(3)  NULL,
    attempts       INT UNSIGNED NOT NULL DEFAULT 0,
    last_error     VARCHAR(512) NULL,
    PRIMARY KEY (event_id),
    KEY idx_identity_outbox_unpublished (published_at, occurred_at)
) ENGINE=InnoDB;

INSERT INTO identity_roles (id, code, name) VALUES
    (1, 'super_admin', '超级管理员'),
    (2, 'department_doctor', '部门医生');

INSERT INTO identity_permissions (id, code, service, name) VALUES
    (1, 'identity.authorization.manage', 'identity', '管理角色、部门归属和账号状态'),
    (2, 'identity.account.manage', 'identity', '管理账号'),
    (3, 'identity.department.manage', 'identity', '管理组织和部门'),
    (4, 'appointment.read', 'appointment', '查看预约'),
    (5, 'appointment.create', 'appointment', '创建预约'),
    (6, 'appointment.update', 'appointment', '修改预约'),
    (7, 'appointment.cancel', 'appointment', '取消预约'),
    (8, 'appointment.reschedule', 'appointment', '改签预约'),
    (9, 'planning.read', 'planning', '查看检查方案'),
    (10, 'planning.create', 'planning', '创建检查方案'),
    (11, 'planning.adjust', 'planning', '人工调整检查方案'),
    (12, 'navigation.read', 'navigation', '查看地图和路线'),
    (13, 'navigation.edit', 'navigation', '编辑地图和地点'),
    (14, 'navigation.publish', 'navigation', '发布地图版本'),
    (15, 'report.read', 'report', '查看报告'),
    (16, 'report.publish', 'report', '发布报告'),
    (17, 'report.correct', 'report', '更正报告'),
    (18, 'report.download', 'report', '下载报告'),
    (19, 'report.export', 'report', '批量导出报告'),
    (20, 'rule.read', 'planning', '查看检查规则'),
    (21, 'rule.edit', 'planning', '编辑检查规则'),
    (22, 'rule.review', 'planning', '审核检查规则'),
    (23, 'rule.publish', 'planning', '发布检查规则');

INSERT INTO identity_role_permissions (role_id, permission_id)
SELECT 1, id FROM identity_permissions;

INSERT INTO identity_role_permissions (role_id, permission_id)
SELECT 2, id
FROM identity_permissions
WHERE service <> 'identity';
