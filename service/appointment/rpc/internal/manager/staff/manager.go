package staff

import (
	"errors"
)

// Manager is the only appointment entry point for administrators and
// authorized department staff. Project, room and schedule operations are
// split into focused files but remain methods of this type.
type Manager struct {
	projectStore ProjectStore
	store        RoomScheduleStore
	bookings     BookingStore
	cache        Cache
	flights      flightGroup
}

func NewManager(projectStore ProjectStore, roomScheduleStore RoomScheduleStore, bookingStore BookingStore, cache Cache) (*Manager, error) {
	if projectStore == nil {
		return nil, errors.New("appointment project store is required")
	}
	if roomScheduleStore == nil {
		return nil, errors.New("appointment room and schedule store is required")
	}
	if bookingStore == nil {
		return nil, errors.New("appointment booking store is required")
	}
	return &Manager{projectStore: projectStore, store: roomScheduleStore, bookings: bookingStore, cache: cache}, nil
}
