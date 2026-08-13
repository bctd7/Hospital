package patient

import (
	"context"
	"time"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager"
)

type (
	Cache             = appointmentmanager.Cache
	RoomScheduleStore = appointmentmanager.RoomScheduleStore
	RoomItem          = appointmentmanager.RoomItem
	ItemWeeklyWindow  = appointmentmanager.ItemWeeklyWindow
	ItemSummary       = appointmentmanager.ItemSummary
	flightGroup       = appointmentmanager.FlightGroup
)

const (
	StatusActive    = appointmentmanager.StatusActive
	hotReadCacheTTL = appointmentmanager.HotReadCacheTTL
	queryCacheTTL   = appointmentmanager.QueryCacheTTL
)

var (
	ErrForbidden = appointmentmanager.ErrForbidden
	ErrNotFound  = appointmentmanager.ErrNotFound
)

func loadCached[T any](ctx context.Context, cache Cache, flights *flightGroup, key string, ttl time.Duration, loader func() (T, bool, error)) (T, bool, error) {
	return appointmentmanager.LoadCached(ctx, cache, flights, key, ttl, loader)
}
