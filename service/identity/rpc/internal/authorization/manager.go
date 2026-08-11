package authorization

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"hospital/common/authn"
	commonauthz "hospital/common/authz"
	contractauthz "hospital/contracts/authz"
)

type Manager struct {
	store Store
}

func NewManager(store Store) (*Manager, error) {
	if store == nil {
		return nil, errors.New("authorization manager store is required")
	}
	return &Manager{store: store}, nil
}

func (m *Manager) GetAuthorizationContext(ctx context.Context, operator authn.Principal, accountID string) (authn.Principal, error) {
	if err := validateID(operator.AccountID, "operator_account_id"); err != nil {
		return authn.Principal{}, err
	}
	if err := validateID(accountID, "account_id"); err != nil {
		return authn.Principal{}, err
	}
	if operator.AccountID != accountID {
		if err := commonauthz.RequirePermission(operator, contractauthz.PermissionIdentityAuthorizationManage); err != nil {
			return authn.Principal{}, fmt.Errorf("%w: %v", ErrForbidden, err)
		}
		return m.store.GetAuthorizationContext(ctx, accountID)
	}
	return operator, nil
}

func validateID(value, field string) error {
	if _, err := uuid.Parse(strings.TrimSpace(value)); err != nil {
		return fmt.Errorf("%w: %s must be a UUID", ErrInvalid, field)
	}
	return nil
}
