package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	account "hospital/service/identity/rpc/internal/authentication"
)

var (
	_ account.Store           = (*Store)(nil)
	_ account.PhoneLoginStore = (*Store)(nil)
)

func (s *Store) FindOrCreateWeChatAccount(ctx context.Context, appID, openID string) (string, error) {
	if appID == "" || openID == "" {
		return "", account.ErrInvalidExternalIdentity
	}
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id
FROM identity_external_identities
WHERE provider = 'wechat' AND provider_app_id = ? AND provider_subject = ?`, appID, openID).Scan(&accountID)
	if err == nil {
		return accountID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("find wechat identity: %w", err)
	}

	accountID = uuid.NewString()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", fmt.Errorf("begin wechat registration: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_accounts (id, account_type, status)
VALUES (?, 'patient', 'active')`, accountID); err != nil {
		return "", fmt.Errorf("create patient account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_external_identities
    (id, account_id, provider, provider_app_id, provider_subject)
VALUES (?, ?, 'wechat', ?, ?)`, uuid.NewString(), accountID, appID, openID); err != nil {
		if isDuplicateEntry(err) {
			_ = tx.Rollback()
			return s.findWeChatAccount(ctx, appID, openID)
		}
		return "", fmt.Errorf("bind wechat identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit wechat registration: %w", err)
	}
	return accountID, nil
}

// FindOrCreateVerifiedPhoneAccount makes the verified phone fingerprint the
// unique login identifier while keeping the immutable UUID as the account key.
func (s *Store) FindOrCreateVerifiedPhoneAccount(ctx context.Context, fingerprint []byte, masked string) (string, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", fmt.Errorf("begin phone registration: %w", err)
	}
	defer tx.Rollback()

	var accountID string
	err = tx.QueryRowContext(ctx, `
SELECT account_id
FROM identity_account_phones
WHERE phone_fingerprint = ?
FOR UPDATE`, fingerprint).Scan(&accountID)
	if err == nil {
		if _, err := tx.ExecContext(ctx, `
UPDATE identity_account_phones
SET phone_masked = ?, verification_status = 'verified', verification_source = 'sms',
    verified_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, masked, accountID); err != nil {
			return "", fmt.Errorf("verify existing phone account: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("commit existing phone login: %w", err)
		}
		return accountID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("find phone account: %w", err)
	}

	accountID = uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_accounts (id, account_type, status)
VALUES (?, 'patient', 'active')`, accountID); err != nil {
		return "", fmt.Errorf("create phone patient account: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_account_phones
    (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at)
VALUES (?, ?, ?, 'verified', 'sms', CURRENT_TIMESTAMP(3))`, accountID, fingerprint, masked); err != nil {
		if isDuplicateEntry(err) {
			_ = tx.Rollback()
			return s.markExistingPhoneVerified(ctx, fingerprint, masked)
		}
		return "", fmt.Errorf("bind verified phone: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit phone registration: %w", err)
	}
	return accountID, nil
}

func (s *Store) markExistingPhoneVerified(ctx context.Context, fingerprint []byte, masked string) (string, error) {
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id
FROM identity_account_phones
WHERE phone_fingerprint = ?`, fingerprint).Scan(&accountID)
	if err != nil {
		return "", fmt.Errorf("find concurrent phone registration: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `
UPDATE identity_account_phones
SET phone_masked = ?, verification_status = 'verified', verification_source = 'sms',
    verified_at = CURRENT_TIMESTAMP(3), updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, masked, accountID); err != nil {
		return "", fmt.Errorf("verify concurrent phone registration: %w", err)
	}
	return accountID, nil
}

func (s *Store) findWeChatAccount(ctx context.Context, appID, openID string) (string, error) {
	var accountID string
	err := s.db.QueryRowContext(ctx, `
SELECT account_id
FROM identity_external_identities
WHERE provider = 'wechat' AND provider_app_id = ? AND provider_subject = ?`, appID, openID).Scan(&accountID)
	if err != nil {
		return "", fmt.Errorf("find concurrent wechat registration: %w", err)
	}
	return accountID, nil
}

func (s *Store) SetSelfReportedPhone(ctx context.Context, accountID string, fingerprint []byte, masked string) (account.PhoneBinding, error) {
	var currentStatus string
	err := s.db.QueryRowContext(ctx, `
SELECT verification_status
FROM identity_account_phones
WHERE account_id = ?`, accountID).Scan(&currentStatus)
	if err == nil && currentStatus == account.PhoneStatusVerified {
		return account.PhoneBinding{}, account.ErrVerifiedPhoneChange
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return account.PhoneBinding{}, fmt.Errorf("read current phone binding: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
UPDATE identity_account_phones
SET phone_fingerprint = ?, phone_masked = ?, verification_status = 'self_reported',
    verification_source = 'self_reported', verified_at = NULL,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE account_id = ?`, fingerprint, masked, accountID)
	if err != nil {
		if isDuplicateEntry(err) {
			return account.PhoneBinding{}, account.ErrPhoneInUse
		}
		return account.PhoneBinding{}, fmt.Errorf("save self-reported phone: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return account.PhoneBinding{}, fmt.Errorf("read phone update result: %w", err)
	}
	if affected == 0 {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `
SELECT EXISTS(SELECT 1 FROM identity_account_phones WHERE account_id = ?)`, accountID).Scan(&exists); err != nil {
			return account.PhoneBinding{}, fmt.Errorf("check existing phone binding: %w", err)
		}
		if exists {
			return account.PhoneBinding{
				Masked: masked, VerificationStatus: account.PhoneStatusSelfReported,
				VerificationSource: account.PhoneSourceSelfReported,
			}, nil
		}
		_, err = s.db.ExecContext(ctx, `
INSERT INTO identity_account_phones
    (account_id, phone_fingerprint, phone_masked, verification_status, verification_source, verified_at)
VALUES (?, ?, ?, 'self_reported', 'self_reported', NULL)`, accountID, fingerprint, masked)
		if err != nil {
			if isDuplicateEntry(err) {
				return account.PhoneBinding{}, account.ErrPhoneInUse
			}
			return account.PhoneBinding{}, fmt.Errorf("insert self-reported phone: %w", err)
		}
	}
	return account.PhoneBinding{
		Masked: masked, VerificationStatus: account.PhoneStatusSelfReported,
		VerificationSource: account.PhoneSourceSelfReported,
	}, nil
}
