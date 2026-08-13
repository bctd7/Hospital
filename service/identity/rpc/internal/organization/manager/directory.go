package manager

import (
	"context"
	"fmt"
	"hospital/service/identity/rpc/internal/organization"
)

type DirectoryManager struct {
	store organization.Store
}

func NewDirectoryManager(store organization.Store) *DirectoryManager {
	return &DirectoryManager{store: store}
}

// GetDirectoryContext returns the public organization entry point. It does
// not require an operator because the plan explicitly exposes this directory
// to unauthenticated visitors as well as authenticated users.
func (m *DirectoryManager) GetDirectoryContext(ctx context.Context) (organization.DirectoryContext, error) {
	active := organization.StatusActive
	hospitals, err := m.store.ListUnits(ctx, organization.ListFilter{Type: organization.UnitTypeHospital, Status: &active})
	if err != nil {
		return organization.DirectoryContext{}, err
	}
	if len(hospitals) != 1 {
		return organization.DirectoryContext{}, fmt.Errorf("%w: expected one active hospital root", organization.ErrDirectoryUnavailable)
	}

	hospital := hospitals[0]
	campuses, err := m.store.ListUnits(ctx, organization.ListFilter{
		Type:     organization.UnitTypeCampus,
		ParentID: &hospital.ID,
		Status:   &active,
	})
	if err != nil {
		return organization.DirectoryContext{}, err
	}
	return organization.DirectoryContext{Hospital: hospital, Campuses: campuses}, nil
}

// ListDirectoryDepartments returns active departments under one active
// campus. Disabled or non-campus parents are hidden as not found so public
// callers cannot enumerate inactive organization structure.
func (m *DirectoryManager) ListDirectoryDepartments(ctx context.Context, campusID string) ([]organization.DirectoryDepartment, error) {
	campusID, err := normalizedUUID(campusID, "campus_id")
	if err != nil {
		return nil, err
	}
	campus, err := m.store.GetUnit(ctx, campusID)
	if err != nil {
		return nil, err
	}
	if campus.Type != organization.UnitTypeCampus || campus.Status != organization.StatusActive {
		return nil, organization.ErrNotFound
	}

	active := organization.StatusActive
	units, err := m.store.ListUnits(ctx, organization.ListFilter{
		Type:     organization.UnitTypeDepartment,
		ParentID: &campusID,
		Status:   &active,
	})
	if err != nil {
		return nil, err
	}
	departments := make([]organization.DirectoryDepartment, 0, len(units))
	for _, unit := range units {
		departments = append(departments, organization.DirectoryDepartment{Unit: unit, CampusName: campus.Name})
	}
	return departments, nil
}
