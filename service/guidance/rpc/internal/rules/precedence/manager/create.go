package manager

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"hospital/common/authn"
	"hospital/service/guidance/rpc/internal/rules/precedence"
)

type CreateInput struct {
	PredecessorItemID string
	SuccessorItemID   string
	StaffReason       string
	PatientMessage    string
	OperationID       string
}

func (m *Manager) Create(ctx context.Context, operator authn.Principal, input CreateInput) (precedence.Rule, error) {
	if err := requireEdit(operator); err != nil {
		return precedence.Rule{}, err
	}
	predecessorID, err := normalizeUUID(input.PredecessorItemID)
	if err != nil {
		return precedence.Rule{}, err
	}
	successorID, err := normalizeUUID(input.SuccessorItemID)
	if err != nil || predecessorID == successorID {
		return precedence.Rule{}, precedence.ErrInvalid
	}
	operationID, err := normalizeUUID(input.OperationID)
	if err != nil {
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

	predecessor, err := m.projects.ResolveProject(ctx, predecessorID)
	if err != nil {
		return precedence.Rule{}, err
	}
	successor, err := m.projects.ResolveProject(ctx, successorID)
	if err != nil {
		return precedence.Rule{}, err
	}
	if err := requireDepartmentScope(operator, successor.DepartmentID); err != nil {
		return precedence.Rule{}, err
	}

	rule := precedence.Rule{
		RuleID:            uuid.NewString(),
		PredecessorItemID: predecessor.ItemID, PredecessorDepartmentID: predecessor.DepartmentID,
		PredecessorItemName: predecessor.Name,
		SuccessorItemID:     successor.ItemID, SuccessorDepartmentID: successor.DepartmentID,
		SuccessorItemName: successor.Name,
		StaffReason:       staffReason, PatientMessage: patientMessage,
		CreatedBy: operator.AccountID, CreateOperationID: operationID,
	}
	var created precedence.Rule
	err = m.store.WithinWriteTransaction(ctx, func(tx precedence.TxStore) error {
		if err := tx.LockGraph(ctx); err != nil {
			return err
		}
		existing, findErr := tx.GetByCreateOperationID(ctx, operationID)
		if findErr == nil {
			if existing.PredecessorItemID != predecessorID || existing.SuccessorItemID != successorID {
				return precedence.ErrConflict
			}
			created = existing
			return nil
		}
		if !errors.Is(findErr, precedence.ErrNotFound) {
			return findErr
		}
		if _, findErr = tx.GetByPair(ctx, predecessorID, successorID); findErr == nil {
			return precedence.ErrConflict
		} else if !errors.Is(findErr, precedence.ErrNotFound) {
			return findErr
		}
		rules, listErr := tx.ListAll(ctx)
		if listErr != nil {
			return listErr
		}
		if reachable(rules, successorID, predecessorID) {
			return precedence.ErrCycle
		}
		created, listErr = tx.Insert(ctx, rule)
		return listErr
	})
	return created, err
}
