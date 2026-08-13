package staff

import "hospital/service/appointment/rpc/internal/manager/common"

type (
	ProjectTxStore         = common.ProjectTxStore
	ProjectOperation       = common.ProjectOperation
	ProjectChange          = common.ProjectChange
	ProjectListFilter      = common.ProjectListFilter
	BookingTxStore         = common.BookingTxStore
	Booking                = common.Booking
	BookingListFilter      = common.BookingListFilter
	BookingOperation       = common.BookingOperation
	BookingOperationChange = common.BookingOperationChange
	BookingStatus          = common.BookingStatus
	DateCapacity           = common.DateCapacity
	BookingSelection       = common.BookingSelection
	ConfigurationTxStore   = common.ConfigurationTxStore
	ConfigurationOperation = common.ConfigurationOperation
	ConfigurationChange    = common.ConfigurationChange
	ExaminationItem        = common.ExaminationItem
	Room                   = common.Room
	RoomItem               = common.RoomItem
	RoomWeeklyWindow       = common.RoomWeeklyWindow
	ItemWeeklyWindow       = common.ItemWeeklyWindow
	ItemSummary            = common.ItemSummary
	Status                 = common.Status
	Session                = common.Session
	Page[T any]            = common.Page[T]
)

const (
	StatusActive           = common.StatusActive
	StatusDisabled         = common.StatusDisabled
	SessionMorning         = common.SessionMorning
	SessionAfternoon       = common.SessionAfternoon
	BookingStatusConfirmed = common.BookingStatusConfirmed
	BookingStatusCheckedIn = common.BookingStatusCheckedIn
)

var (
	ErrInvalid            = common.ErrInvalid
	ErrForbidden          = common.ErrForbidden
	ErrNotFound           = common.ErrNotFound
	ErrConflict           = common.ErrConflict
	ErrVersionConflict    = common.ErrVersionConflict
	ErrInvalidState       = common.ErrInvalidState
	ErrWindowConflict     = common.ErrWindowConflict
	ErrNotImplemented     = common.ErrNotImplemented
	FormatRoomDisplayName = common.FormatRoomDisplayName
)
