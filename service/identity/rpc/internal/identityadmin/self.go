package identityadmin

import (
	"context"

	"hospital/common/authn"
)

// GetDisplayProfile returns the profile owned by the current signed-in account.
func (m *Manager) GetDisplayProfile(ctx context.Context, operator authn.Principal) (DisplayProfile, error) {
	if err := validUUID(operator.AccountID, "account_id"); err != nil {
		return DisplayProfile{}, ErrForbidden
	}
	return m.store.GetDisplayProfile(ctx, operator.AccountID)
}

// UpdateDisplayProfile changes only the profile owned by the current signed-in account.
func (m *Manager) UpdateDisplayProfile(ctx context.Context, operator authn.Principal, nickname string) (DisplayProfile, error) {
	if err := validUUID(operator.AccountID, "account_id"); err != nil {
		return DisplayProfile{}, ErrForbidden
	}
	nickname, err := normalizedText(nickname, "nickname", maxNicknameRunes, true)
	if err != nil {
		return DisplayProfile{}, err
	}
	return m.store.UpdateDisplayProfile(ctx, operator.AccountID, nickname)
}
