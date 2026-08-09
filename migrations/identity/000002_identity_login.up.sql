USE hospital_identity;

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
