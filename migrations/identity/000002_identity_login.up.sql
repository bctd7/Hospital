USE hospital_identity;

-- 外部登录身份：把微信等供应商身份映射到平台账号；不保存登录 code、AppSecret 或 session_key。
-- 首期业务只使用一个微信身份，但独立表避免把微信 OpenID 当成平台账号主键，并保留多登录来源扩展边界。
CREATE TABLE identity_external_identities (
    id                   CHAR(36)     NOT NULL,
    account_id           CHAR(36)     NOT NULL,
    provider             VARCHAR(32)  NOT NULL,
    provider_app_id      VARCHAR(64)  NOT NULL,
    provider_subject     VARCHAR(128) NOT NULL,
    created_at           DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at           DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (id),
    UNIQUE KEY uk_identity_external_subject (provider, provider_app_id, provider_subject),
    KEY idx_identity_external_account (account_id),
    CONSTRAINT fk_identity_external_account FOREIGN KEY (account_id) REFERENCES identity_accounts (id),
    CONSTRAINT chk_identity_external_provider CHECK (provider IN ('wechat'))
) ENGINE=InnoDB;

-- 当前手机号绑定：account_id 是主键，因此一个账号首期最多一个手机号。
-- 独立存放是因为手机号具有独立的验证状态和安全策略；数据库不保存手机号明文。
CREATE TABLE identity_account_phones (
    account_id          CHAR(36)    NOT NULL,
    phone_fingerprint   BINARY(32)  NOT NULL,
    phone_masked        VARCHAR(32) NOT NULL,
    verification_status VARCHAR(16) NOT NULL DEFAULT 'self_reported',
    verification_source VARCHAR(16) NOT NULL DEFAULT 'self_reported',
    verified_at         DATETIME(3) NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (account_id),
    UNIQUE KEY uk_identity_account_phones_fingerprint (phone_fingerprint),
    CONSTRAINT fk_identity_phone_account FOREIGN KEY (account_id) REFERENCES identity_accounts (id),
    CONSTRAINT chk_identity_phone_status CHECK (verification_status IN ('self_reported', 'verified')),
    CONSTRAINT chk_identity_phone_source CHECK (verification_source IN ('self_reported', 'sms', 'wechat', 'admin'))
) ENGINE=InnoDB;
