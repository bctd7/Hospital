package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"hospital/service/identity/rpc/internal/account"
)

func (s *Store) GetDisplayProfile(ctx context.Context, accountID string) (account.DisplayProfile, error) {
	var profile account.DisplayProfile
	err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(ap.nickname, ''), a.management_version
FROM identity_accounts a
LEFT JOIN identity_account_profiles ap ON ap.account_id = a.id
WHERE a.id = ?`, accountID).Scan(&profile.Nickname, &profile.ManagementVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return account.DisplayProfile{}, account.ErrNotFound
	}
	if err != nil {
		return account.DisplayProfile{}, fmt.Errorf("get account display profile: %w", err)
	}
	return profile, nil
}

func (s *Store) UpdateDisplayProfile(ctx context.Context, accountID, nickname string) (account.DisplayProfile, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return account.DisplayProfile{}, fmt.Errorf("begin display profile update: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
UPDATE identity_accounts
SET management_version = management_version + 1,
    updated_at = CURRENT_TIMESTAMP(3)
WHERE id = ?`, accountID)
	if err != nil {
		return account.DisplayProfile{}, fmt.Errorf("bump display profile version: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return account.DisplayProfile{}, fmt.Errorf("read display profile update result: %w", err)
	}
	if affected == 0 {
		return account.DisplayProfile{}, account.ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO identity_account_profiles (account_id, nickname)
VALUES (?, NULLIF(?, ''))
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), updated_at = CURRENT_TIMESTAMP(3)`, accountID, nickname); err != nil {
		return account.DisplayProfile{}, fmt.Errorf("save account display profile: %w", err)
	}
	var profile account.DisplayProfile
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(ap.nickname, ''), a.management_version
FROM identity_accounts a
LEFT JOIN identity_account_profiles ap ON ap.account_id = a.id
WHERE a.id = ?`, accountID).Scan(&profile.Nickname, &profile.ManagementVersion); err != nil {
		return account.DisplayProfile{}, fmt.Errorf("read updated display profile: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return account.DisplayProfile{}, fmt.Errorf("commit display profile update: %w", err)
	}
	return profile, nil
}
