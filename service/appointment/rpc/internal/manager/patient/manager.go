package patient

import (
	"errors"
)

// Manager is the only patient-facing appointment entry point. Booking
// and capacity operations will be added here without leaking into staff flows.
type Manager struct {
	store   RoomScheduleStore
	cache   Cache
	flights flightGroup
}

func NewManager(store RoomScheduleStore, cache Cache) (*Manager, error) {
	if store == nil {
		return nil, errors.New("patient appointment store is required")
	}
	return &Manager{store: store, cache: cache}, nil
}
