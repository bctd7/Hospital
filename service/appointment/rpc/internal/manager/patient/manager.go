package patient

import (
	"errors"
)

// Manager is the only patient-facing appointment entry point. Booking
// and capacity operations will be added here without leaking into staff flows.
type Manager struct {
	projects ProjectStore
	store    RoomScheduleStore
	bookings BookingStore
	cache    Cache
	flights  flightGroup
}

func NewManager(projects ProjectStore, store RoomScheduleStore, bookings BookingStore, cache Cache) (*Manager, error) {
	if projects == nil {
		return nil, errors.New("patient examination project store is required")
	}
	if store == nil {
		return nil, errors.New("patient appointment store is required")
	}
	if bookings == nil {
		return nil, errors.New("patient booking store is required")
	}
	return &Manager{projects: projects, store: store, bookings: bookings, cache: cache}, nil
}
