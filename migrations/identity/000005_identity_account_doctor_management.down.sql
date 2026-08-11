ALTER TABLE identity_staff_profiles
    DROP CHECK chk_identity_staff_status,
    DROP INDEX idx_identity_staff_directory,
    DROP COLUMN staff_status,
    DROP COLUMN description,
    DROP COLUMN avatar_url,
    DROP COLUMN display_name;

DROP TABLE IF EXISTS identity_account_profiles;

ALTER TABLE identity_accounts
    DROP CHECK chk_identity_accounts_management_version,
    DROP COLUMN management_version;
