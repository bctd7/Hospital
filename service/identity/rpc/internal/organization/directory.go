package organization

import (
	"context"
	"fmt"
)

// GetDirectoryContext returns the public organization entry point. It does
// not require an operator because the plan explicitly exposes this directory
// to unauthenticated visitors as well as authenticated users.
func (m *Manager) GetDirectoryContext(ctx context.Context) (DirectoryContext, error) {
	active := StatusActive
	hospitals, err := m.listUnits(ctx, ListFilter{Type: UnitTypeHospital, Status: &active})
	if err != nil {
		return DirectoryContext{}, err
	}
	if len(hospitals) != 1 {
		return DirectoryContext{}, fmt.Errorf("%w: expected one active hospital root", ErrDirectoryUnavailable)
	}

	hospital := hospitals[0]
	campuses, err := m.listUnits(ctx, ListFilter{
		Type:     UnitTypeCampus,
		ParentID: &hospital.ID,
		Status:   &active,
	})
	if err != nil {
		return DirectoryContext{}, err
	}
	return DirectoryContext{Hospital: hospital, Campuses: campuses}, nil
}

// ListDirectoryDepartments returns active departments under one active
// campus. Disabled or non-campus parents are hidden as not found so public
// callers cannot enumerate inactive organization structure.
func (m *Manager) ListDirectoryDepartments(ctx context.Context, campusID string) ([]DirectoryDepartment, error) {
	campusID, err := normalizedUUID(campusID, "campus_id")
	if err != nil {
		return nil, err
	}
	campus, err := m.store.GetUnit(ctx, campusID)
	if err != nil {
		return nil, err
	}
	if campus.Type != UnitTypeCampus || campus.Status != StatusActive {
		return nil, ErrNotFound
	}

	active := StatusActive
	units, err := m.listUnits(ctx, ListFilter{
		Type:     UnitTypeDepartment,
		ParentID: &campusID,
		Status:   &active,
	})
	if err != nil {
		return nil, err
	}
	departments := make([]DirectoryDepartment, 0, len(units))
	for _, unit := range units {
		departments = append(departments, DirectoryDepartment{Unit: unit, CampusName: campus.Name})
	}
	return departments, nil
}
