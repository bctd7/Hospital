package precedence

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"hospital/common/authn"
)

type CreateInput struct {
	PredecessorItemID string
	SuccessorItemID   string
	StaffReason       string
	PatientMessage    string
	OperationID       string
}

func (m *Manager) Create(ctx context.Context, operator authn.Principal, input CreateInput) (Rule, error) {
	if err := requireEdit(operator); err != nil {
		return Rule{}, err
	}
	predecessorID, err := normalizeUUID(input.PredecessorItemID)
	if err != nil {
		return Rule{}, err
	}
	successorID, err := normalizeUUID(input.SuccessorItemID)
	if err != nil || predecessorID == successorID {
		return Rule{}, ErrInvalid
	}
	operationID, err := normalizeUUID(input.OperationID)
	if err != nil {
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

	predecessor, err := m.projects.ResolveProject(ctx, predecessorID)
	if err != nil {
		return Rule{}, err
	}
	successor, err := m.projects.ResolveProject(ctx, successorID)
	if err != nil {
		return Rule{}, err
	}
	if err := requireDepartmentScope(operator, successor.DepartmentID); err != nil {
		return Rule{}, err
	}

	rule := Rule{
		RuleID: uuid.NewString(), OwnerItemID: successor.ItemID,
		PredecessorItemID: predecessor.ItemID, PredecessorDepartmentID: predecessor.DepartmentID,
		PredecessorItemName: predecessor.Name,
		SuccessorItemID:     successor.ItemID, SuccessorDepartmentID: successor.DepartmentID,
		SuccessorItemName: successor.Name,
		StaffReason:       staffReason, PatientMessage: patientMessage,
		CreatedBy: operator.AccountID, CreateOperationID: operationID,
	}
	var created Rule
	err = m.store.WithinWriteTransaction(ctx, func(tx TxStore) error {
		if err := tx.LockGraph(ctx); err != nil {
			return err
		}
		existing, findErr := tx.GetByCreateOperationID(ctx, operationID)
		if findErr == nil {
			if existing.PredecessorItemID != predecessorID || existing.SuccessorItemID != successorID {
				return ErrConflict
			}
			created = existing
			return nil
		}
		if !errors.Is(findErr, ErrNotFound) {
			return findErr
		}
		if _, findErr = tx.GetByPair(ctx, predecessorID, successorID); findErr == nil {
			return ErrConflict
		} else if !errors.Is(findErr, ErrNotFound) {
			return findErr
		}
		rules, listErr := tx.ListAll(ctx)
		if listErr != nil {
			return listErr
		}
		if reachable(rules, successorID, predecessorID) {
			return ErrCycle
		}
		created, listErr = tx.Insert(ctx, rule)
		return listErr
	})
	return created, err
}
