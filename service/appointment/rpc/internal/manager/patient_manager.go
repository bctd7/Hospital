package manager

import (
	"errors"
)

// PatientManager is the only patient-facing appointment entry point. Booking
// and capacity operations will be added here without leaking into staff flows.
type PatientManager struct {
	store   RoomScheduleStore
	cache   Cache
	flights flightGroup
}

func NewPatientManager(store RoomScheduleStore, cache Cache) (*PatientManager, error) {
	if store == nil {
		return nil, errors.New("patient appointment store is required")
	}
	return &PatientManager{store: store, cache: cache}, nil
}
