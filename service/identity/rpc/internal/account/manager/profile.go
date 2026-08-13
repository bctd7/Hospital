package manager

import (
	"context"

	"hospital/common/authn"
	"hospital/service/identity/rpc/internal/account"
)

// GetDisplayProfile returns the profile owned by the current signed-in account.
func (m *Manager) GetDisplayProfile(ctx context.Context, operator authn.Principal) (account.DisplayProfile, error) {
	if err := validUUID(operator.AccountID, "account_id"); err != nil {
		return account.DisplayProfile{}, account.ErrForbidden
	}
	return m.store.GetDisplayProfile(ctx, operator.AccountID)
}

// UpdateDisplayProfile changes only the profile owned by the current signed-in account.
func (m *Manager) UpdateDisplayProfile(ctx context.Context, operator authn.Principal, nickname string) (account.DisplayProfile, error) {
	if err := validUUID(operator.AccountID, "account_id"); err != nil {
		return account.DisplayProfile{}, account.ErrForbidden
	}
	nickname, err := normalizedText(nickname, "nickname", maxNicknameRunes, true)
	if err != nil {
		return account.DisplayProfile{}, err
	}
	return m.store.UpdateDisplayProfile(ctx, operator.AccountID, nickname)
}
