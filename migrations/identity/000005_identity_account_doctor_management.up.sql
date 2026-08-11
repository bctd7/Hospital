ALTER TABLE identity_accounts
    ADD COLUMN management_version BIGINT UNSIGNED NOT NULL DEFAULT 1
        AFTER authorization_version,
    ADD CONSTRAINT chk_identity_accounts_management_version
        CHECK (management_version > 0);

CREATE TABLE identity_account_profiles (
    account_id CHAR(36)     NOT NULL,
    nickname   VARCHAR(64)  NULL,
    created_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
        ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (account_id),
    KEY idx_identity_account_profiles_nickname (nickname, account_id),
    CONSTRAINT fk_identity_account_profiles_account
        FOREIGN KEY (account_id) REFERENCES identity_accounts (id)
) ENGINE=InnoDB;

ALTER TABLE identity_staff_profiles
    ADD COLUMN display_name VARCHAR(128) NOT NULL DEFAULT '' AFTER staff_no,
    ADD COLUMN avatar_url   VARCHAR(2048) NULL AFTER display_name,
    ADD COLUMN description  VARCHAR(512) NULL AFTER avatar_url,
    ADD COLUMN staff_status VARCHAR(16) NOT NULL DEFAULT 'active' AFTER description,
    ADD KEY idx_identity_staff_directory
        (department_id, staff_status, display_name, staff_no, account_id),
    ADD CONSTRAINT chk_identity_staff_status
        CHECK (staff_status IN ('active', 'revoked'));

UPDATE identity_staff_profiles
SET display_name = COALESCE(NULLIF(staff_no, ''), '医生')
WHERE display_name = '';
