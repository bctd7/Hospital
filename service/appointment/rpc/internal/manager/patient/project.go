package patient

import (
	"context"
	"errors"
	"fmt"

	"hospital/common/authn"
)

// ListAvailableRooms returns the rooms a patient may choose for a project.
// The selected room remains a patient choice and is not assigned by staff.
func (m *Manager) ListAvailableRooms(ctx context.Context, patient authn.Principal, itemID string) ([]RoomItem, error) {
	if err := requirePatient(patient); err != nil {
		return nil, err
	}
	itemID, err := normalizeUUID(itemID, "item_id")
	if err != nil {
		return nil, err
	}
	item, err := m.getItemSummary(ctx, itemID)
	if err != nil || item.Status != StatusActive {
		return nil, ErrNotFound
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:rooms:true", item.DepartmentID, m.generation(ctx, item.DepartmentID), itemID)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, queryCacheTTL, func() ([]RoomItem, bool, error) {
		value, loadErr := m.store.ListItemRooms(ctx, itemID, true)
		return value, true, loadErr
	})
	return result, err
}

// ListProjectWindows exposes only active project windows to patients.
func (m *Manager) ListProjectWindows(ctx context.Context, patient authn.Principal, itemID string) ([]ItemWeeklyWindow, error) {
	if err := requirePatient(patient); err != nil {
		return nil, err
	}
	itemID, err := normalizeUUID(itemID, "item_id")
	if err != nil {
		return nil, err
	}
	item, err := m.getItemSummary(ctx, itemID)
	if err != nil || item.Status != StatusActive {
		return nil, ErrNotFound
	}
	key := fmt.Sprintf("department:%s:g:%s:item:%s:windows:true", item.DepartmentID, m.generation(ctx, item.DepartmentID), itemID)
	result, _, err := loadCached(ctx, m.cache, &m.flights, key, hotReadCacheTTL, func() ([]ItemWeeklyWindow, bool, error) {
		value, loadErr := m.store.ListItemWindows(ctx, itemID, true)
		return value, true, loadErr
	})
	return result, err
}

func requirePatient(principal authn.Principal) error {
	if principal.AccountType != authn.AccountTypePatient || principal.AccountID == "" {
		return ErrForbidden
	}
	if _, err := normalizeUUID(principal.AccountID, "patient account_id"); err != nil {
		return ErrForbidden
	}
	return nil
}

func (m *Manager) generation(ctx context.Context, departmentID string) string {
	if m.cache == nil {
		return "0"
	}
	value, err := m.cache.DepartmentGeneration(ctx, departmentID)
	if err != nil {
		return "0"
	}
	return value
}

func (m *Manager) getItemSummary(ctx context.Context, itemID string) (ItemSummary, error) {
	value, found, err := loadCached(ctx, m.cache, &m.flights, "item-summary:"+itemID, hotReadCacheTTL, func() (ItemSummary, bool, error) {
		item, loadErr := m.store.GetItemSummary(ctx, itemID)
		if errors.Is(loadErr, ErrNotFound) {
			return ItemSummary{}, false, nil
		}
		return item, loadErr == nil, loadErr
	})
	if err != nil {
		return ItemSummary{}, err
	}
	if !found {
		return ItemSummary{}, ErrNotFound
	}
	return value, nil
}
