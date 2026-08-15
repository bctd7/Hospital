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
	BookingListView        = common.BookingListView
	ReportTxStore          = common.ReportTxStore
	ExaminationReport      = common.ExaminationReport
	ReportVersion          = common.ReportVersion
	ReportContent          = common.ReportContent
	ReportListFilter       = common.ReportListFilter
	ReportTemplate         = common.ReportTemplate
	ReportStatus           = common.ReportStatus
	ReportVersionStatus    = common.ReportVersionStatus
	ReportVersionKind      = common.ReportVersionKind
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
	StatusActive                 = common.StatusActive
	StatusDisabled               = common.StatusDisabled
	SessionMorning               = common.SessionMorning
	SessionAfternoon             = common.SessionAfternoon
	BookingStatusConfirmed       = common.BookingStatusConfirmed
	BookingStatusInProgress      = common.BookingStatusInProgress
	BookingStatusCompleted       = common.BookingStatusCompleted
	BookingStatusNoShow          = common.BookingStatusNoShow
	BookingStatusCanceled        = common.BookingStatusCanceled
	BookingListViewActive        = common.BookingListViewActive
	BookingListViewCompleted     = common.BookingListViewCompleted
	ReportStatusDraft            = common.ReportStatusDraft
	ReportStatusPublished        = common.ReportStatusPublished
	ReportVersionStatusDraft     = common.ReportVersionStatusDraft
	ReportVersionStatusPublished = common.ReportVersionStatusPublished
	ReportVersionKindInitial     = common.ReportVersionKindInitial
	ReportVersionKindCorrection  = common.ReportVersionKindCorrection
)

var (
	ErrInvalid                 = common.ErrInvalid
	ErrForbidden               = common.ErrForbidden
	ErrNotFound                = common.ErrNotFound
	ErrConflict                = common.ErrConflict
	ErrVersionConflict         = common.ErrVersionConflict
	ErrInvalidState            = common.ErrInvalidState
	ErrExaminationWindowClosed = common.ErrExaminationWindowClosed
	ErrWindowConflict          = common.ErrWindowConflict
	ErrNotImplemented          = common.ErrNotImplemented
	FormatRoomDisplayName      = common.FormatRoomDisplayName
)
