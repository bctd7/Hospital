package manager

import (
	"context"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type UpdateInput struct {
	RuleID          string
	StaffReason     string
	PatientMessage  string
	ExpectedVersion int64
	OperationID     string
}

func (m *Manager) Update(ctx context.Context, operator authn.Principal, input UpdateInput) (precedence.Rule, error) {
	if err := requireEdit(operator); err != nil {
		return precedence.Rule{}, err
	}
	ruleID, err := normalizeUUID(input.RuleID)
	if err != nil || input.ExpectedVersion < 1 {
		return precedence.Rule{}, precedence.ErrInvalid
	}
	if _, err = normalizeUUID(input.OperationID); err != nil {
		return precedence.Rule{}, err
	}
	staffReason, err := normalizeRequiredText(input.StaffReason, 512)
	if err != nil {
		return precedence.Rule{}, err
	}
	patientMessage, err := normalizeOptionalText(input.PatientMessage, 512)
	if err != nil {
		return precedence.Rule{}, err
	}
	var updated precedence.Rule
	err = m.store.WithinWriteTransaction(ctx, func(tx precedence.TxStore) error {
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
