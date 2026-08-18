package precedence

import (
	"context"

	"hospital/common/authn"
)

type UpdateInput struct {
	RuleID          string
	StaffReason     string
	PatientMessage  string
	ExpectedVersion int64
	OperationID     string
}

func (m *Manager) Update(ctx context.Context, operator authn.Principal, input UpdateInput) (Rule, error) {
	if err := requireEdit(operator); err != nil {
		return Rule{}, err
	}
	ruleID, err := normalizeUUID(input.RuleID)
	if err != nil || input.ExpectedVersion < 1 {
		return Rule{}, ErrInvalid
	}
	if _, err = normalizeUUID(input.OperationID); err != nil {
		return Rule{}, err
	}
	staffReason, err := normalizeRequiredText(input.StaffReason, 512)
	if err != nil {
		return Rule{}, err
	}
	patientMessage, err := normalizeOptionalText(input.PatientMessage, 512)
	if err != nil {
		return Rule{}, err
	}
	var updated Rule
	err = m.store.WithinWriteTransaction(ctx, func(tx TxStore) error {
		rule, getErr := tx.GetByIDForUpdate(ctx, ruleID)
		if getErr != nil {
			return getErr
		}
		if scopeErr := requireDepartmentScope(operator, rule.SuccessorDepartmentID); scopeErr != nil {
			return scopeErr
		}
		updated, getErr = tx.UpdateText(ctx, ruleID, staffReason, patientMessage, input.ExpectedVersion)
		return getErr
	})
	return updated, err
}
