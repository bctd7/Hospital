package precedence

import (
	"context"

	"hospital/common/authn"
)

type DeleteInput struct {
	RuleID          string
	ExpectedVersion int64
	OperationID     string
}

func (m *Manager) Delete(ctx context.Context, operator authn.Principal, input DeleteInput) error {
	if err := requireEdit(operator); err != nil {
		return err
	}
	ruleID, err := normalizeUUID(input.RuleID)
	if err != nil || input.ExpectedVersion < 1 {
		return ErrInvalid
	}
	if _, err = normalizeUUID(input.OperationID); err != nil {
		return err
	}
	return m.store.WithinWriteTransaction(ctx, func(tx TxStore) error {
		if lockErr := tx.LockGraph(ctx); lockErr != nil {
			return lockErr
		}
		rule, getErr := tx.GetByIDForUpdate(ctx, ruleID)
		if getErr != nil {
			return getErr
		}
		if scopeErr := requireDepartmentScope(operator, rule.SuccessorDepartmentID); scopeErr != nil {
			return scopeErr
		}
		return tx.Delete(ctx, ruleID, input.ExpectedVersion)
	})
}
