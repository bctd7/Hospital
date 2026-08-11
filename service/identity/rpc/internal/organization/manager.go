package organization

import (
	"context"
	"fmt"
)

const (
	ActionUnitCreated  = "identity.organization.unit.created"
	ActionUnitUpdated  = "identity.organization.unit.updated"
	ActionUnitDisabled = "identity.organization.unit.disabled"
	ActionUnitEnabled  = "identity.organization.unit.enabled"
)

type CreateUnitCommand struct {
	Type        UnitType
	ParentID    string
	Name        string
	OperationID string
	RequestID   string
}

type UpdateUnitCommand struct {
	UnitID          string
	Name            *string
	ParentID        *string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type ChangeUnitStatusCommand struct {
	UnitID          string
	ExpectedVersion int64
	OperationID     string
	RequestID       string
}

type Manager struct {
	store Store
}

func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

// getUnit is the package-internal raw read used by the public directory and
// administrator-facing methods after they apply their own access rules.
func (m *Manager) getUnit(ctx context.Context, unitID string) (Unit, error) {
	unitID, err := normalizedUUID(unitID, "unit_id")
	if err != nil {
		return Unit{}, err
	}
	return m.store.GetUnit(ctx, unitID)
}

// listUnits is the package-internal flat query. Callers outside this package
// must use either the public directory or administrator-facing methods.
func (m *Manager) listUnits(ctx context.Context, filter ListFilter) ([]Unit, error) {
	if !filter.Type.Valid() {
		return nil, fmt.Errorf("%w: unsupported unit_type %q", ErrInvalid, filter.Type)
	}
	if filter.ParentID != nil {
		parentID, err := normalizedUUID(*filter.ParentID, "parent_id")
		if err != nil {
			return nil, err
		}
		filter.ParentID = &parentID
	}
	if filter.Status != nil && !filter.Status.Valid() {
		return nil, fmt.Errorf("%w: unsupported status %q", ErrInvalid, *filter.Status)
	}
	return m.store.ListUnits(ctx, filter)
}
