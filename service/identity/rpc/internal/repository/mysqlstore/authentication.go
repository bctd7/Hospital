package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"hospital/service/identity/rpc/internal/authentication"
)

var _ authentication.Store = (*Store)(nil)

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
