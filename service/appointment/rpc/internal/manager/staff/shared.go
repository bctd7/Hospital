package staff

import (
	"context"
	"time"

	appointmentmanager "hospital/service/appointment/rpc/internal/manager"
)

type (
	Cache                 = appointmentmanager.Cache
	ProjectStore          = appointmentmanager.ProjectStore
	ProjectTxStore        = appointmentmanager.ProjectTxStore
	ProjectOperation      = appointmentmanager.ProjectOperation
	ProjectChange         = appointmentmanager.ProjectChange
	ProjectListFilter     = appointmentmanager.ProjectListFilter
	RoomScheduleStore     = appointmentmanager.RoomScheduleStore
	RoomScheduleTxStore   = appointmentmanager.RoomScheduleTxStore
	RoomScheduleOperation = appointmentmanager.RoomScheduleOperation
	RoomScheduleChange    = appointmentmanager.RoomScheduleChange
	ExaminationItem       = appointmentmanager.ExaminationItem
	Room                  = appointmentmanager.Room
	RoomItem              = appointmentmanager.RoomItem
	RoomWeeklyWindow      = appointmentmanager.RoomWeeklyWindow
	ItemWeeklyWindow      = appointmentmanager.ItemWeeklyWindow
	ItemSummary           = appointmentmanager.ItemSummary
	Status                = appointmentmanager.Status
	Session               = appointmentmanager.Session
	Page[T any]           = appointmentmanager.Page[T]
	flightGroup           = appointmentmanager.FlightGroup
)

const (
	StatusActive     = appointmentmanager.StatusActive
	StatusDisabled   = appointmentmanager.StatusDisabled
	SessionMorning   = appointmentmanager.SessionMorning
	SessionAfternoon = appointmentmanager.SessionAfternoon
	hotReadCacheTTL  = appointmentmanager.HotReadCacheTTL
	queryCacheTTL    = appointmentmanager.QueryCacheTTL
)

func loadCached[T any](ctx context.Context, cache Cache, flights *flightGroup, key string, ttl time.Duration, loader func() (T, bool, error)) (T, bool, error) {
	return appointmentmanager.LoadCached(ctx, cache, flights, key, ttl, loader)
}

func jitteredTTL(base time.Duration) time.Duration { return appointmentmanager.JitteredTTL(base) }

var (
	ErrInvalid            = appointmentmanager.ErrInvalid
	ErrForbidden          = appointmentmanager.ErrForbidden
	ErrNotFound           = appointmentmanager.ErrNotFound
	ErrConflict           = appointmentmanager.ErrConflict
	ErrVersionConflict    = appointmentmanager.ErrVersionConflict
	ErrInvalidState       = appointmentmanager.ErrInvalidState
	ErrWindowConflict     = appointmentmanager.ErrWindowConflict
	ErrNotImplemented     = appointmentmanager.ErrNotImplemented
	FormatRoomDisplayName = appointmentmanager.FormatRoomDisplayName
)
